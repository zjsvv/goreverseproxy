# Configuration Documentation

This project utilizes a YAML-based configuration file that defines various settings for the reverse proxy. Below is an explanation of each key in the configuration and an example.

## Configuration Keys

### 1. `targetUrl`
- **Description**: The base URL to which the reverse proxy will forward requests.
- **Example**: `"http://localhost"`

### 2. `targetPort`
- **Description**: The port on the target server that the reverse proxy will communicate with.
- **Example**: `"9000"`

### 3. `blockedHeaders`
- **Description**: A list of HTTP headers that are blocked from being forwarded to the target server. These headers are filtered out for security or privacy purposes.
- **Example**:
  ```yaml
  blockedHeaders:
    - "X-Custom-Key"
    - "Accesstoken"
  ```
### 4. `blockedQueryParams`
- **Description**: A list of query parameters that should not be forwarded to the target server. These are typically sensitive parameters.
- **Example**:
  ```yaml
  blockedQueryParams:
    - "filter"
    - "category"
  ```

### 5. `maskedNeededKeys`
- **Description**: A list of keys in the response body that need to be masked for privacy or compliance. The value of keys will be replaced with masked values(`*`) with the same length of the value.
- **Example**:
  ```yaml
  maskedNeededKeys:
  - "address"
  - "password"
  ```

## Example Configuration
```yaml
targetUrl: "http://localhost"
targetPort: "9000"

blockedHeaders:
  - "X-Custom-Key"
  - "Accesstoken"

blockedQueryParams:
  - "filter"
  - "category"

maskedNeededKeys:
  - "creditcard"
  - "phonenumber"

```
This configuration forwards requests to http://localhost:9000, blocks specific headers and query parameters, and masks sensitive information in the response body.
