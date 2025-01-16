import json
import requests
from helpers import (
    parse_args,
    get_logger,
    get_login_token,
    initialize_clients,
    ensure_bucket_exists,
    get_all_instances,
    get_all_compartments,
)

logger = get_logger()


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
    token, token_err = get_login_token(
        username=username,
        password=password,
        verify_cert=verify_cert,
        management_node_name=management_node_name,
        management_node_vip=management_node_vip,
    )
    if token_err:
        return token_err  # pass the error up

    export_url = f"https://{management_node_vip if management_node_vip else management_node_name}/20160918/instances/{instance_ocid}/actions/export"
    headers = {"Content-Type": "application/json", "Authorization": f"Bearer {token}"}
    payload = {
        "bucketName": bucket_name,
        "destinationType": "objectStorageTuple",
        "namespaceName": namespace_name,
        "compartmentId": bucket_compartment_ocid,
    }

    try:
        resp = requests.post(
            export_url, headers=headers, json=payload, verify=verify_cert
        )
        resp.raise_for_status()
        logger.info(
            f"Export initiated for instance {instance_ocid}, status={resp.status_code}"
        )
        return None  # No error
    except requests.exceptions.RequestException as e:
        return RuntimeError(f"Failed to export {instance_ocid}: {e}")


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
    """
    Return error or None.
    """
    logger.info(f"Processing host: {host_name} (IP={host_ip})")
    for tenancy_ocid in tenancy_list:
        comp_ids, comp_err = get_all_compartments(identity_client, tenancy_ocid)
        if comp_err:
            return comp_err

        # get namespace
        namespace_name = object_storage_client.get_namespace().data

        # ensure bucket
        bucket_comp_ocid, bucket_err = ensure_bucket_exists(
            bucket_name, namespace_name, tenancy_ocid, object_storage_client
        )
        if bucket_err:
            return bucket_err

        for cid in comp_ids:
            inst_ids, inst_err = get_all_instances(compute_client, cid)
            if inst_err:
                return inst_err

            for instance_ocid in inst_ids:
                err_http = export_instance_backup_http(
                    instance_ocid=instance_ocid,
                    bucket_name=bucket_name,
                    namespace_name=namespace_name,
                    management_node_name=host_name,
                    management_node_vip=host_ip,
                    bucket_compartment_ocid=bucket_comp_ocid,
                    username=username,
                    password=password,
                    verify_cert=verify_cert,
                )
                if err_http:
                    return err_http
    return None  # success


def execute_all(
    hosts,
    compute_client,
    object_storage_client,
    identity_client,
    username,
    password,
    verify_cert=False,
):
    for h in hosts:
        err = process_host(
            host_ip=h["HOST_IP"],
            host_name=h["HOST_NAME"],
            tenancy_list=h["TENANCY_LIST"],
            bucket_name=h["BUCKET_NAME"],
            compute_client=compute_client,
            object_storage_client=object_storage_client,
            identity_client=identity_client,
            username=username,
            password=password,
            verify_cert=verify_cert,
        )
        if err:
            return err
    return None


def execute_instance(
    target_host, object_storage_client, username, password, verify_cert=False
):
    host_ip = target_host["HOST_IP"]
    host_name = target_host["HOST_NAME"]
    tenancy_ocid = target_host["TENANCY_OCID"]
    bucket_name = target_host["BUCKET_NAME"]
    instance_ocid = target_host["INSTANCE_OCID"]

    namespace_name = object_storage_client.get_namespace().data

    bucket_comp_ocid, bucket_err = ensure_bucket_exists(
        bucket_name, namespace_name, tenancy_ocid, object_storage_client
    )
    if bucket_err:
        return bucket_err

    logger.info(f"Processing single instance: {instance_ocid} on host {host_name}")
    err_http = export_instance_backup_http(
        instance_ocid=instance_ocid,
        bucket_name=bucket_name,
        namespace_name=namespace_name,
        management_node_vip=host_ip,
        management_node_name=host_name,
        bucket_compartment_ocid=bucket_comp_ocid,
        username=username,
        password=password,
        verify_cert=verify_cert,
    )
    if err_http:
        return err_http

    logger.info(f"Backup completed for instance {instance_ocid} on host {host_name}")
    return None  # success


def main():
    args = parse_args()

    # Log the CLI arguments
    logger.info("=== CLI Arguments ===")
    logger.info(f"Mode : {args.mode}")
    logger.info(f"Hosts JSON    : {args.hosts}")
    logger.info(f"Target Host   : {args.target_host}")
    logger.info("====================")

    # Convert JSON if provided
    hosts = json.loads(args.hosts) if args.hosts else None
    target_host = json.loads(args.target_host) if args.target_host else None

    # Init clients
    c, compute_client, obj_storage_client, identity_client, init_err = (
        initialize_clients(
            config_path=args.oci_config, profile_name=args.oci_config_profile_name
        )
    )
    if init_err:
        logger.error(f"Could not initialize OCI clients: {init_err}")
        raise init_err

    # Dispatch
    if args.mode == "all":
        err = execute_all(
            hosts=hosts,
            compute_client=compute_client,
            object_storage_client=obj_storage_client,
            identity_client=identity_client,
            username=args.username,
            password=args.password,
        )
        if err:
            logger.error(f"Backup failed in 'all' mode: {err}")
            raise err
    elif args.mode == "instance":
        err = execute_instance(
            target_host=target_host,
            object_storage_client=obj_storage_client,
            username=args.username,
            password=args.password,
        )
        if err:
            logger.error(f"Backup failed in 'instance' mode: {err}")
            raise err
    else:
        logger.error(f"Invalid mode: {args.mode}")
        raise ValueError("Invalid mode")

    logger.info("Backup process completed successfully.")


if __name__ == "__main__":
    main()
