# Instance Backup Script

This script allows you to automate the process of exporting instance backups to an Object Storage bucket in Oracle Cloud Infrastructure (OCI). It supports processing multiple hosts and tenancies or targeting specific instances for backup.

---

## Features
- Export backups for all instances within specified compartments and tenancies.
- Target specific instances for backup.
- Supports multiple hosts and configurations.

---

## Prerequisites

1. **Python Environment**:
   - Python 3.7+
   - Required libraries:
     - `oci`
     - `requests`
     - `logging`
   - Install dependencies using pip:
     ```bash
     pip install oci requests
     ```

2. **OCI Configuration**:
   - Create an OCI config file (e.g., `~/.oci/config`) containing credentials for API access. For more details, refer to the [OCI Configuration Documentation](https://docs.oracle.com/en-us/iaas/Content/API/Concepts/sdkconfig.htm).

3. **JSON Input File**:
   - The script requires a `data.json` file that specifies configuration details such as hosts, tenancies, and buckets. Refer to the [Input File Structure](#input-file-structure) section.

4. **Network Setup**:
   - Ensure that the host(s) can access the management node VIP over the appropriate network.

---

## Input File Structure

The `data.json` file provides the necessary configuration for the script.

### Example `data.json`
```json
{
    "Mode": "all",
    "OCI_CONFIG_PATH": "/home/ali/Documents/config/xcbgde/config.txt",
    "HOSTS": [
        {
            "HOST_IP": "192.168.1.1",
            "HOST_NAME": "host1",
            "TENANCY_LIST": [
                "ocid1.tenancy.oc1..exampleuniqueID1",
                "ocid1.tenancy.oc1..exampleuniqueID2"
            ],
            "BUCKET_NAME": "bucket1"
        },
        {
            "HOST_IP": "192.168.1.2",
            "HOST_NAME": "host2",
            "TENANCY_LIST": [
                "ocid1.tenancy.oc1..exampleuniqueID3"
            ],
            "BUCKET_NAME": "bucket2"
        }
    ],
    "TARGET_INSTANCE": "instance1",
    "TARGET_HOST": {
        "HOST_IP": "192.168.1.1",
        "HOST_NAME": "host1",
        "BUCKET_COMPARTMENT_OCID": "ocid1.bucket.oc1..exampleuniqueID1",
        "BUCKET_NAME": "bucket1"
    }
}
```

### Key Fields
- **Mode**: Specifies the mode of operation (`all` for all hosts or `instance` for a specific instance).
- **OCI_CONFIG_PATH**: Path to the OCI config file.
- **HOSTS**: List of host configurations.
  - **HOST_IP**: Management node VIP.
  - **HOST_NAME**: Name of the host.
  - **TENANCY_LIST**: List of tenancy OCIDs.
  - **BUCKET_NAME**: Object Storage bucket name.
- **TARGET_INSTANCE**: ID of the instance to back up (used in `instance` mode).
- **TARGET_HOST**: Details of the host when targeting a single instance.
  - **BUCKET_COMPARTMENT_OCID**: OCID of the bucket compartment.
  - **BUCKET_NAME**: Object Storage bucket name.

---

## How to Run the Script

1. Clone the repository or download the script.

2. Place your configuration file (e.g., `config.txt`) in the appropriate directory.

3. Create a `data.json` file with your configuration details.

4. Execute the script:
   ```bash
   python main.py
   ```

5. The script will initiate the backup process based on the mode specified in `data.json`.

---

## Output 

```
2024-12-16 22:24:02,935 [INFO] OCI clients initialized successfully.
2024-12-16 22:24:02,971 [INFO] Processing host: host1 with IP 192.168.1.1
2024-12-16 22:24:05,777 [INFO] Bucket 'bucket1' already exists in namespace 'axq3ds'.
2024-12-16 22:24:07,011 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaamugmoyj7un
2024-12-16 22:24:08,218 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaatgi5rhz4q7kp6ldblu6sehbyttq.
2024-12-16 22:24:09,436 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaalp4lx6tbdwahjpyhoaqvvskpacytssuojrkbkssza.
2024-12-16 22:24:10,687 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaaqavjwtaxvnm64oxa4zmgriqauchk2qga.
2024-12-16 22:24:11,894 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaaax3w3z7ku3xr5wy3xg2ez7gw5qjnkijicqa.
2024-12-16 22:24:13,107 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaaq4xiqjzxqa7ocqhaf7fpubgbnskzfognlrdsm3q.
2024-12-16 22:24:14,321 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaa5kki3dfth2uclkynblbvxhho7qcl7gdakha.
2024-12-16 22:24:15,610 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaamcognwn3wdemzrz437w2op27h7oeqbdddybewyq.
2024-12-16 22:24:16,833 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaagwn3mthjzelbgbvnlvz4f75xlp4pxmzjiwq.
2024-12-16 22:24:18,082 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaar5xlcjc4q7ksjj4tf7cgzxbrcaybu5gnkitna.
2024-12-16 22:24:19,332 [INFO] Found 1 running instances in compartment ocid1.compartment.oc1..aaaaaaaa5xsobulcmmxbzowazte2fldgfnra.
2024-12-16 22:24:20,595 [INFO] Found 0 running instances in compartment ocid1.compartment.oc1..aaaaaaaagzdo6ot6yvxue73gqvjy3tyvqfb7epa.
2024-12-16 22:24:21,927 [INFO] Found 3 running instances in compartment ocid1.compartment.oc1..aaaaaaaa7hhnaygv7tvcmcspa2algla4w5njpaq.
2024-12-16 22:24:23,097 [INFO] Found 0 running instances in compartment ocid1.tenancy.oc1..aaaaaaaaqtnxtv77pftojgv4hglpf6fnva.
2024-12-16 22:24:23,097 [INFO] Backup process completed successfully.
```


## Logging
The script outputs logs to the console for easy monitoring. Logs include:
- Processing details for hosts and instances.
- Status updates for backup exports.
- Errors and exceptions.

---

## Troubleshooting
- **Invalid Mode**: Ensure the `Mode` field in `data.json` is set to either `all` or `instance`.
- **Authentication Errors**: Verify your OCI config file and credentials.
- **Network Issues**: Check network connectivity to the management node VIP.

---

## Contribution
Feel free to open issues or submit pull requests to improve this script.

---

