# network-connection-check

The `network-connection-check` validates TCP/UDP connectivity to a target address. It reports success when the connection behavior matches the expected reachability.

## Configuration

Set these environment variables in the `HealthCheck` spec:

- `CONNECTION_TARGET` (required): target address to dial. Accepts `tcp://` or `udp://` prefixes (for example, `tcp://github.com:443`).
- `CONNECTION_TARGET_UNREACHABLE` (optional): set to `true` when the target is expected to be unreachable.
- `KUBECONFIG` (optional): explicit kubeconfig path for local development.

The check timeout defaults to 20 seconds but is overridden by the Kuberhealthy run deadline when available.

## Build

- `just build` builds the container image locally.
- `just test` runs unit tests.
- `just binary` builds the binary in `bin/`.

## Example HealthCheck

Apply the example below or the provided `healthcheck.yaml`:

```yaml
apiVersion: kuberhealthy.github.io/v2
kind: HealthCheck
metadata:
  name: kuberhealthy-github-reachable
  namespace: kuberhealthy
spec:
  runInterval: 30m
  timeout: 10m
  podSpec:
    spec:
      containers:
        - name: kuberhealthy-github-reachable
          image: kuberhealthy/network-connection-check:sha-<short-sha>
          imagePullPolicy: IfNotPresent
          env:
            - name: CONNECTION_TARGET
              value: "tcp://github.com:443"
      restartPolicy: Never
```
