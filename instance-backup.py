import oci
import os
import requests
import logging
from helpers import *

# Configure Logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[logging.StreamHandler()],
)
logger = logging.getLogger(__name__)


# Export instance backup using HTTP Client
def export_instance_backup_http(
    instance_id,
    bucket_name,
    namespace_name,
    bucket_compartment_id,
    management_node_vip,
    host_name,
    username,
    password,
    tenancy="pcan01",
    verify_cert=False
):
    """
    Export instance backup using PCA's local token-based authentication.
    """
    # 1. Get a login token
    token = get_login_token(
        management_node_vip=management_node_vip,
        username=username,
        password=password,
        tenancy=tenancy,
        verify_cert=verify_cert
    )
    
    export_url = f"https://iaas.pcan01.sherwin.com/20160918/instances/{instance_id}/actions/export"
    
    # 3. Build request headers with Bearer token
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {token}"
    }
    
    # 4. Request payload for the export
    payload = {
        "bucketName": bucket_name,
        "destinationType": "objectStorageTuple",
        "namespaceName": namespace_name,
        "compartmentId": bucket_compartment_id,
    }

    try:
        response = requests.post(
            export_url,
            headers=headers,
            json=payload,
            verify=verify_cert  # either False or path to CA cert
        )
        response.raise_for_status()
        logger.info(f"Export initiated for instance {instance_id} with status code {response.status_code}.")
    except requests.exceptions.RequestException as e:
        logger.error(f"Failed to export instance {instance_id}: {e}")
        raise


# Process a single host
def process_host(
    host_ip,
    host_name,
    tenancy_list,
    bucket_name,
    compute_client,
    object_storage_client,
    identity_client,
    signer,
):

    try:
        logger.info(f"Processing host: {host_name} with IP {host_ip}")

        for tenancy_ocid in tenancy_list:
            compartment_ids = get_all_compartments(identity_client, tenancy_ocid)
            # Fetch namespace name for the tenancy
            namespace_name = object_storage_client.get_namespace().data
            # Ensure bucket exists
            bucket_compartment_id = ensure_bucket_exists(
                bucket_name, namespace_name, tenancy_ocid, object_storage_client
            )
            for compartment_id in compartment_ids:
                # Fetch all running instances
                instance_ids = get_all_instances(compute_client, compartment_id)
                for instance_id in instance_ids:
                    # Export backups for all instances
                    # https://docs.oracle.com/en-us/iaas/compute-cloud-at-customer/topics/compute/creating-an-instance-backup.htm
                    export_instance_backup_http(
                        bucket_name=bucket_name,
                        bucket_compartment_id=bucket_compartment_id,
                        instance_id=instance_id,
                        management_node_vip=host_ip,
                        namespace_name=namespace_name,
                        host_name=host_name,
                        signer=signer,
                    )

    except Exception as e:
        logger.error(f"Failed to process host {host_name} with IP {host_ip}: {e}")
        raise


# Process a single instance
def process_single_instance(
    instance_id,
    bucket_name,
    namespace_name,
    bucket_compartment_id,
    host_ip,
    host_name,
    signer,
):

    try:
        logger.info(f"Processing single instance: {instance_id}")
        export_instance_backup_http(
            instance_id=instance_id,
            bucket_name=bucket_name,
            namespace_name=namespace_name,
            management_node_vip=host_ip,
            host_name=host_name,
            bucket_compartment_id=bucket_compartment_id,
            username="afs881",           
            password="gvg!kga.tya4uvr*YZP", 
            tenancy="pcan01",             # or your PCA tenancy name
            verify_cert=False             # or "/root/.oci/pcan01.ca.crt" if you have a valid CA
        )
        logger.info(f"Backup process completed for instance {instance_id}.")
    except Exception as e:
        logger.error(f"Failed to process instance {instance_id}: {e}")
        raise


# Main logic
# Dispatch table for modes
def execute_all(
    hosts,
    compute_client,
    object_storage_client,
    identity_client,
    signer,
):
    for host in hosts:
        process_host(
            bucket_name=host["BUCKET_NAME"],
            host_ip=host["HOST_IP"],
            host_name=host["HOST_NAME"],
            tenancy_list=host["TENANCY_LIST"],
            compute_client=compute_client,
            object_storage_client=object_storage_client,
            identity_client=identity_client,
            signer=signer,
        )


def execute_instance(
    target_instance,
    target_host,
    object_storage_client,
    signer,
):
    host_ip = target_host["HOST_IP"]
    bucket_name = target_host["BUCKET_NAME"]
    host_name = target_host["HOST_NAME"]
    namespace_name = object_storage_client.get_namespace().data
    bucket_compartment_id = target_host["BUCKET_COMPARTMENT_OCID"]

    process_single_instance(
        instance_id=target_instance,
        bucket_name=bucket_name,
        namespace_name=namespace_name,
        host_ip=host_ip,
        host_name=host_name,
        bucket_compartment_id=bucket_compartment_id,
        signer=signer,
    )


# Main dispatch table
def main():

    # PCA site tenancy limits: https://docs.oracle.com/en/engineered-systems/private-cloud-appliance/3.0-latest/relnotes/relnotes-limits-config.html

    logger = get_logger()
    # Load json data
    json_file_path = "data.json"
    data = load_json(json_file_path)

    # Load values
    mode = data.get("Mode", "all")  # Modes: "all", "instance"
    CONFIG_PATH = data.get("OCI_CONFIG_PATH", "~/.oci/config")
    hosts = data.get("HOSTS", [])
    target_instance = data.get("TARGET_INSTANCE", None)
    target_host = data.get("TARGET_HOST", None)

    # Initialize OCI clients
    config, compute_client, object_storage_client, identity_client = initialize_clients(
        CONFIG_PATH
    )
    signer = initialize_signer(config_path=CONFIG_PATH)

    # Dispatch table
    dispatch = {
        "all": lambda: execute_all(
            hosts, compute_client, object_storage_client, identity_client, signer=signer
        ),
        "instance": lambda: execute_instance(
            target_instance, target_host, object_storage_client, signer=signer
        ),
    }

    # Execute based on mode
    try:
        if mode in dispatch:
            dispatch[mode]()
            logger.info("Backup process completed successfully.")
        else:
            logger.error(f"Invalid mode specified: {mode}")
    except Exception as e:
        logger.error(f"Backup process failed: {e}")
        raise


if __name__ == "__main__":
    main()
