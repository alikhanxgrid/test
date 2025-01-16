import argparse
import logging
import oci
from oci import Signer
import json
import requests
import re


def get_logger():
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(message)s",
        handlers=[logging.StreamHandler()],
    )
    return logging.getLogger(__name__)


def get_tenancy_name(management_node_name: str) -> str:
    """
    Returns whatever is between "iaas." and ".sherwin" in the input string.
    Returns None if no match is found.
    """
    # Regex explanation:
    # ^iaas\.       matches the prefix "iaas."
    # ([^.]+)       captures one or more characters that are not a period
    # \.sherwin     matches ".sherwin"
    pattern = r"^iaas\.([^.]+)\.sherwin"
    match = re.search(pattern, management_node_name)
    if match:
        return match.group(1)
    else:
        raise ValueError(f"Failed to extract tenancy name from {management_node_name}.")


def parse_args():
    parser = argparse.ArgumentParser(description="Run Oracle PCA instance backups.")
    parser.add_argument(
        "--mode",
        required=True,
        choices=["all", "instance"],
        help="Which mode to run the script in: 'all' or 'instance'",
    )
    parser.add_argument(
        "--oci-config",
        required=True,
        help="Path to the OCI config file (e.g., '~/.oci/config').",
    )
    parser.add_argument(
        "--hosts", default=None, help="JSON string of HOSTS array (used if mode='all')."
    )
    parser.add_argument(
        "--target-host",
        default=None,
        help="JSON string of TARGET_HOST object (used if mode='instance').",
    )
    parser.add_argument(
        "--username",
        required=False,
        help="PCA username for token-based authentication.",
    )
    parser.add_argument(
        "--oci-config-profile-name",
        required=False,
        help="OCI profile name for authentication.",
    )
    parser.add_argument(
        "--password",
        required=False,
        help="PCA tenancy password for authentication.",
    )
    return parser.parse_args()


def get_login_token(
    username: str,
    password: str,
    management_node_name,
    verify_cert=False,
    management_node_vip=None,
) -> str:
    """
    Logs in to PCA's local authentication endpoint to obtain a Bearer token.
    """
    login_url = f"https://{management_node_vip if management_node_vip else management_node_name}/20160918/login"

    payload = {
        "username": username,
        "password": password,
        "tenancy": get_tenancy_name(management_node_name),
    }
    try:
        response = requests.post(login_url, json=payload, verify=verify_cert)
        response.raise_for_status()
        data = response.json()
        return data["token"]
    except requests.exceptions.RequestException as e:
        raise RuntimeError(f"Failed to get login token from {login_url}: {e}")


# Initialize OCI Config and Clients
def initialize_clients(config_path, profile_name):
    try:
        logger = get_logger()
        config = oci.config.from_file(
            file_location=config_path, profile_name=profile_name
        )
        compute_client = oci.core.ComputeClient(config)
        object_storage_client = oci.object_storage.ObjectStorageClient(config)
        identity_client = oci.identity.IdentityClient(config)
        compute_client.base_client.session.verify = False
        object_storage_client.base_client.session.verify = False
        identity_client.base_client.session.verify = False
        logger.info("OCI clients initialized successfully.")
        return config, compute_client, object_storage_client, identity_client
    except Exception as e:
        logger.error(f"Failed to initialize OCI clients: {e}")
        raise


def initialize_signer(config_path):
    config = oci.config.from_file(config_path, profile_name="pcan01")
    signer = Signer(
        tenancy=config["tenancy"],
        user=config["user"],
        fingerprint=config["fingerprint"],
        private_key_file_location=config["key_file"],
        pass_phrase=oci.config.get_config_value_or_default(config, "pass_phrase"),
    )
    return signer


def ensure_bucket_exists(
    bucket_name,
    namespace_name,
    compartment_id,
    object_storage_client: oci.object_storage.ObjectStorageClient,
):
    try:
        logger = get_logger()
        # Check if the bucket exists
        bucket_details = object_storage_client.get_bucket(namespace_name, bucket_name)
        logger.info(
            f"Bucket '{bucket_name}' already exists in namespace '{namespace_name}'."
        )

        return bucket_details.data.compartment_id
    except oci.exceptions.ServiceError as e:
        if e.status == 404:
            # Bucket does not exist, create it
            logger.info(f"Bucket '{bucket_name}' does not exist. Creating it...")
            create_details = oci.object_storage.models.CreateBucketDetails(
                name=bucket_name,
                compartment_id=compartment_id,
                public_access_type="ObjectRead",
            )
            bucket_details: oci.Response[oci.object_storage.models.Bucket] = (
                object_storage_client.create_bucket(namespace_name, create_details)
            )
            logger.info(f"Bucket '{bucket_name}' created successfully.")
            return bucket_details.data.compartment_id
        else:
            logger.error(f"Failed to check/create bucket '{bucket_name}': {e}")
            raise


# Fetch all instances for a tenancy
def get_all_instances(compute_client: oci.core.ComputeClient, compartment_id):
    try:
        logger = get_logger()
        instances = []
        response = compute_client.list_instances(compartment_id)
        for instance in response.data:
            instances.append(instance.id)
        logger.info(
            f"Found {len(instances)} running instances in compartment {compartment_id}."
        )
        return instances
    except Exception as e:
        logger.error(f"Failed to fetch instances for compartment {compartment_id}: {e}")
        raise


def get_all_compartments(
    identity_client: oci.identity.IdentityClient, tenancy_ocid
) -> list:
    logger = get_logger()
    try:
        compartments: oci.Response[list[oci.identity.models.Compartment]] = (
            identity_client.list_compartments(
                compartment_id=tenancy_ocid,
                access_level="ANY",
                compartment_id_in_subtree=True,
                lifecycle_state="ACTIVE",
            )
        )
        data = compartments.data
        compartment_ids = [item.id for item in data]
        compartment_ids.append(tenancy_ocid)
        return compartment_ids
    except Exception as e:
        logger.error(f"Failed to fetch compartments for tenancy {tenancy_ocid}: {e}")
        raise


def load_json(json_file_path):
    """
    Load configuration from a JSON file.
    """
    try:
        with open(json_file_path, "r") as file:
            return json.load(file)
    except FileNotFoundError:
        raise Exception(f"Configuration file {json_file_path} not found.")
    except json.JSONDecodeError as e:
        raise Exception(f"Error decoding JSON file {json_file_path}: {e}")
