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
    instance_ocid,
    bucket_name,
    namespace_name,
    management_node_name,
    bucket_compartment_ocid,
    username,
    password,
    verify_cert,
    management_node_vip=None,
):
    """
    Export instance backup using PCA's local token-based authentication.
    """
    # 1. Get a login token
    token = get_login_token(
        username=username,
        password=password,
        verify_cert=verify_cert,
        management_node_name=management_node_name,
        management_node_vip=management_node_vip,
    )
    export_url = f"https://{management_node_vip if management_node_vip else management_node_name}/20160918/instances/{instance_ocid}/actions/export"
    # 3. Build request headers with Bearer token
    headers = {"Content-Type": "application/json", "Authorization": f"Bearer {token}"}

    # 4. Request payload for the export
    payload = {
        "bucketName": bucket_name,
        "destinationType": "objectStorageTuple",
        "namespaceName": namespace_name,
        "compartmentId": bucket_compartment_ocid,
    }

    try:
        response = requests.post(
            export_url,
            headers=headers,
            json=payload,
            verify=verify_cert,  # either False or path to CA cert
        )
        response.raise_for_status()
        logger.info(
            f"Export initiated for instance {instance_ocid} with status code {response.status_code}."
        )
    except requests.exceptions.RequestException as e:
        logger.error(f"Failed to export instance {instance_ocid}: {e}")
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
    username,
    password,
    verify_cert,
):

    try:
        logger.info(f"Processing host: {host_name} with IP {host_ip}")

        for tenancy_ocid in tenancy_list:
            compartment_ids = get_all_compartments(identity_client, tenancy_ocid)
            # Fetch namespace name for the tenancy
            namespace_name = object_storage_client.get_namespace().data
            # Ensure bucket exists
            bucket_compartment_ocid = ensure_bucket_exists(
                bucket_name, namespace_name, tenancy_ocid, object_storage_client
            )
            for compartment_id in compartment_ids:
                # Fetch all running instances
                instance_ocids = get_all_instances(compute_client, compartment_id)
                for instance_ocid in instance_ocids:
                    # Export backups for all instances
                    # https://docs.oracle.com/en-us/iaas/compute-cloud-at-customer/topics/compute/creating-an-instance-backup.htm
                    export_instance_backup_http(
                        management_node_name=host_name,
                        management_node_vip=host_ip,
                        instance_ocid=instance_ocid,
                        bucket_compartment_ocid=bucket_compartment_ocid,
                        bucket_name=bucket_name,
                        namespace_name=namespace_name,
                        username=username,
                        password=password,
                        verify_cert=verify_cert,
                    )

    except Exception as e:
        logger.error(f"Failed to process host {host_name} with IP {host_ip}: {e}")
        raise


# Main logic
# Dispatch table for modes
def execute_all(
    hosts,
    compute_client,
    object_storage_client,
    identity_client,
    username,
    password,
    verify_cert=False,
):
    for host in hosts:
        process_host(
            host_ip=host["HOST_IP"],
            host_name=host["HOST_NAME"],
            tenancy_list=host["TENANCY_LIST"],
            bucket_name=host["BUCKET_NAME"],
            compute_client=compute_client,
            object_storage_client=object_storage_client,
            identity_client=identity_client,
            username=username,
            password=password,
            verify_cert=verify_cert,
        )


def execute_instance(
    target_host,
    object_storage_client,
    username,
    password,
    verify_cert=False,
):
    host_ip = target_host["HOST_IP"]
    host_name = target_host["HOST_NAME"]
    tenancy_ocid = target_host["TENANCY_OCID"]
    bucket_name = target_host["BUCKET_NAME"]
    namespace_name = object_storage_client.get_namespace().data
    instance_ocid = target_host["INSTANCE_OCID"]

    bucket_compartment_ocid = ensure_bucket_exists(
        bucket_name, namespace_name, tenancy_ocid, object_storage_client
    )

    try:
        logger.info(f"Processing single instance: {instance_ocid} on host {host_name}")
        export_instance_backup_http(
            instance_ocid=instance_ocid,
            bucket_name=bucket_name,
            namespace_name=namespace_name,
            management_node_vip=host_ip,
            management_node_name=host_name,
            bucket_compartment_ocid=bucket_compartment_ocid,
            username=username,
            password=password,
            verify_cert=verify_cert,
        )
        logger.info(
            f"Backup process completed for instance {instance_ocid} on host {host_name}"
        )
    except Exception as e:
        logger.error(
            f"Failed to process instance {instance_ocid} on host {host_name} {e}"
        )
        raise


# Main dispatch table
def main():

    # PCA site tenancy limits: https://docs.oracle.com/en/engineered-systems/private-cloud-appliance/3.0-latest/relnotes/relnotes-limits-config.html

    logger = get_logger()
    args = parse_args()

    # --------------------------------------------------------
    # Print/log the values that were passed in from the workflow
    # --------------------------------------------------------
    logger.info("=== CLI Arguments Received ===")
    logger.info(f"Mode            : {args.mode}")
    logger.info(f"OCI Config Path : {args.oci_config}")
    logger.info(f"Hosts (JSON)    : {args.hosts}")
    logger.info(f"Target Host JSON: {args.target_host}")
    logger.info(f"Username        : {len(args.username)}")
    logger.info(f"Profile Name    : {len(args.oci_config_profile_name)}")
    logger.info("==============================")

    # Load values
    mode = args.mode  # Modes: "all", "instance"
    CONFIG_PATH = args.oci_config
    if args.hosts is not None:
        hosts = json.loads(args.hosts)
    if args.target_host is not None:
        target_host = json.loads(args.target_host)
    username = args.username
    profile_name = args.oci_config_profile_name
    password = args.password

    # Initialize OCI clients
    config, compute_client, object_storage_client, identity_client = initialize_clients(
        config_path=CONFIG_PATH, profile_name=profile_name
    )
    # signer = initialize_signer(config_path=CONFIG_PATH)

    # Dispatch table
    dispatch = {
        "all": lambda: execute_all(
            compute_client=compute_client,
            object_storage_client=object_storage_client,
            identity_client=identity_client,
            hosts=hosts,
            username=username,
            password=password,
        ),
        "instance": lambda: execute_instance(
            target_host=target_host,
            object_storage_client=object_storage_client,
            username=username,
            password=password,
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
