#!/bin/bash
# Monitor k3d cluster resources during tests
# Run this in the background to capture resource usage over time

set -euo pipefail

INTERVAL="${1:-30}"  # Check every N seconds
DURATION="${2:-300}" # Run for N seconds (5 minutes)
OUTPUT="${3:-resource-monitor.log}"

echo "Starting resource monitoring (interval=${INTERVAL}s, duration=${DURATION}s)" > "$OUTPUT"
echo "================================================================" >> "$OUTPUT"

END_TIME=$(($(date +%s) + DURATION))

while [ $(date +%s) -lt $END_TIME ]; do
    TIMESTAMP=$(date '+%Y-%m-%d %H:%M:%S')
    
    echo "" >> "$OUTPUT"
    echo "=== $TIMESTAMP ===" >> "$OUTPUT"
    
    # Docker disk usage
    echo "--- Docker Disk Usage ---" >> "$OUTPUT"
    docker system df >> "$OUTPUT" 2>&1 || true
    
    # k3d node disk usage
    echo "--- k3d Node Disk Usage ---" >> "$OUTPUT"
    docker exec k3d-k3s-default-server-0 df -h 2>> "$OUTPUT" | \
        grep -E '(Filesystem|/$|/var/lib|overlay)' >> "$OUTPUT" || true
    
    # PVC counts
    echo "--- PVC Status ---" >> "$OUTPUT"
    kubectl get pvc --all-namespaces -o json 2>/dev/null | \
        jq -r '.items | group_by(.status.phase) | map({phase: .[0].status.phase, count: length}) | .[]' >> "$OUTPUT" 2>&1 || true
    
    # Local path provisioner status
    echo "--- Provisioner Pods ---" >> "$OUTPUT"
    kubectl get pods -n kube-system -l app=local-path-provisioner -o wide >> "$OUTPUT" 2>&1 || true
    
    sleep "$INTERVAL"
done

echo "Resource monitoring complete" >> "$OUTPUT"
