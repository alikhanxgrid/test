# File: helpers.py

import argparse
import logging
import oci
import requests
import re


def get_logger():
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(message)s",
        handlers=[logging.StreamHandler()],
    )
    return logging.getLogger(__name__)


logger = get_logger()


def get_tenancy_name(management_node_name: str):
    pattern = r"^iaas\.([^.]+)\.sherwin"
    match = re.search(pattern, management_node_name)
    if match:
        return match.group(1)
    # Instead of raising here, return None or a specific error
    return None


def parse_args():
    parser = argparse.ArgumentParser(description="Run Oracle PCA instance backups.")
    parser.add_argument("--mode", required=True, choices=["all", "instance"])
    parser.add_argument("--oci-config", required=True)
    parser.add_argument("--hosts", default=None)
    parser.add_argument("--target-host", default=None)
    parser.add_argument("--username", required=False)
    parser.add_argument("--oci-config-profile-name", required=False)
    parser.add_argument("--password", required=False)
    return parser.parse_args()


def get_login_token(
    username,
    password,
    management_node_name,
    verify_cert=False,
    management_node_vip=None,
):
    """
    Returns (token_string, error), where:
      token_string = valid token if success
      error = None if success, otherwise an Exception object
    """
    login_url = f"https://{management_node_vip if management_node_vip else management_node_name}/20160918/login"

    tenancy_name = get_tenancy_name(management_node_name)
    if not tenancy_name:
        return None, ValueError(
            f"Could not parse tenancy from '{management_node_name}'"
        )

    payload = {
        "username": username,
        "password": password,
        "tenancy": tenancy_name,
    }

    try:
        resp = requests.post(login_url, json=payload, verify=verify_cert)
        resp.raise_for_status()
        data = resp.json()
        return data["token"], None
    except requests.exceptions.RequestException as e:
        return None, RuntimeError(f"Failed to get login token from {login_url}: {e}")


def initialize_clients(config_path, profile_name):
    """
    Returns (config, compute_client, object_storage_client, identity_client, error).
      If success, error = None.
      If fail, error = exception object describing what went wrong.
    """
    try:
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
        return config, compute_client, object_storage_client, identity_client, None
    except Exception as e:
        return None, None, None, None, e


def ensure_bucket_exists(
    bucket_name, namespace_name, compartment_id, object_storage_client
):
    """
    Returns (bucket_compartment_id, error).
      If success, error=None.
      If fail, error=exception object.
    """
    # Minimal logging for info, but no error logs:
    try:
        bucket_details = object_storage_client.get_bucket(namespace_name, bucket_name)
        logger.info(f"Bucket '{bucket_name}' found in namespace '{namespace_name}'.")
        return bucket_details.data.compartment_id, None
    except oci.exceptions.ServiceError as e:
        if e.status == 404:
            # Bucket doesn't exist -> create
            logger.info(f"Bucket '{bucket_name}' not found. Creating it...")
            create_details = oci.object_storage.models.CreateBucketDetails(
                name=bucket_name,
                compartment_id=compartment_id,
                public_access_type="ObjectRead",
            )
            try:
                result = object_storage_client.create_bucket(
                    namespace_name, create_details
                )
                logger.info(f"Bucket '{bucket_name}' created.")
                return result.data.compartment_id, None
            except Exception as create_err:
                return None, create_err
        else:
            return None, e


def get_all_instances(compute_client, compartment_id):
    """
    Returns (list_of_instance_ids, error).
    """
    try:
        instances = []
        response = compute_client.list_instances(compartment_id)
        for inst in response.data:
            instances.append(inst.id)
        logger.info(f"Found {len(instances)} running instances in {compartment_id}.")
        return instances, None
    except Exception as e:
        return None, e


def get_all_compartments(identity_client, tenancy_ocid):
    """
    Returns (list_of_compartment_ids, error).
    """
    try:
        resp = identity_client.list_compartments(
            compartment_id=tenancy_ocid,
            access_level="ANY",
            compartment_id_in_subtree=True,
            lifecycle_state="ACTIVE",
        )
        data = resp.data
        comp_ids = [c.id for c in data]
        comp_ids.append(tenancy_ocid)
        return comp_ids, None
    except Exception as e:
        return None, e
