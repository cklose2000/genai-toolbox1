# Workflow Automation Tools for Complex Multi-Service Pipelines

## Problem Statement

Users struggle with creating complex multi-step workflows in GCP, particularly:
- **SFTP → GCS → ETL → Email** pipelines
- Each service connection requires different authentication methods
- Error handling across service boundaries is complex
- Claude Code can handle individual services but struggles with the orchestration

## Root Cause

The main challenge isn't the individual services (Claude Code handles those well), but the **connection points** between services:
- **SFTP → GCS**: Requires agent pools, SSH keys, network configuration
- **GCS → ETL**: Needs event triggers, file detection, format handling
- **ETL → Email**: Requires completion detection, error aggregation, formatting

## Proposed Solution: Connection Templates

Instead of complex service wrappers, create a simple MCP tool that provides connection templates.

### Tool: `workflow-connector`

A single tool that returns the exact gcloud commands needed to connect any two services:

```yaml
tools:
  workflow_connector:
    kind: workflow-connector
    source: bigquery-source  # Reuse existing auth
    description: "Get commands to connect workflow services"
    parameters:
      - name: from
        description: "Source service (sftp, gcs, bigquery, etc.)"
      - name: to
        description: "Target service (gcs, bigquery, email, etc.)"
```

### Example Usage

User: "Connect SFTP to my data pipeline with notifications"

Claude Code would:
1. Call `workflow-connector` with `from: sftp, to: gcs`
2. Get agent pool setup commands with placeholders
3. Fill in user's specific values
4. Execute the commands
5. Repeat for each connection in the pipeline

### Implementation Sketch

```go
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
    connections := map[string]string{
        "sftp-to-gcs": sftpToGCSTemplate,
        "gcs-to-etl": gcsToETLTemplate,
        "etl-to-email": etlToEmailTemplate,
    }
    
    key := fmt.Sprintf("%s-to-%s", params.GetString("from"), params.GetString("to"))
    return []any{connections[key]}, nil
}
```

## Template Examples

### SFTP to GCS Connection
```bash
# 1. Create agent pool for SFTP
gcloud transfer agent-pools create {{.PoolName}} \
  --project={{.Project}} \
  --display-name="SFTP Agent Pool"

# 2. Install agent on a VM that can reach SFTP
curl -O https://storage.googleapis.com/cloud-transfer-service/latest/install.sh
sudo bash install.sh --agent-pool={{.PoolName}}

# 3. Create transfer job
gcloud transfer jobs create {{.JobName}} \
  --project={{.Project}} \
  --source-agent-pool={{.PoolName}} \
  --source-path={{.SFTPPath}} \
  --destination-bucket={{.Bucket}} \
  --schedule="{{.Schedule}}"
```

### GCS to ETL Connection
```bash
# Trigger ETL when files arrive
gcloud functions deploy trigger-etl \
  --trigger-resource {{.Bucket}} \
  --trigger-event google.storage.object.finalize \
  --entry-point handleFileUpload \
  --set-env-vars DATASET={{.Dataset}},TABLE={{.Table}}
```

### ETL to Email Connection
```bash
# Add completion notification to ETL
gcloud workflows deploy etl-with-notification \
  --source=- <<EOF
main:
  steps:
    - runETL:
        call: http.post
        args:
          url: {{.ETLFunction}}
    - notify:
        call: http.post
        args:
          url: {{.EmailFunction}}
          body:
            status: \${runETL.body.status}
EOF
```

## Benefits

1. **Minimal abstraction** - Just returns gcloud commands
2. **Solves the real pain point** - Connection complexity
3. **Flexible** - Claude Code can adapt commands as needed
4. **Educational** - Users see exactly what's happening

## Alternative: Recipe Documentation

Even simpler approach - just add markdown files:
```
/recipes/
├── sftp-to-gcs.md
├── gcs-to-bigquery-etl.md
└── workflow-orchestration.md
```

Then a single tool that returns these recipes when asked.

## Next Steps

1. Validate with more use cases
2. Identify most common connection patterns
3. Create minimal MVP with 3-4 connection types
4. Test with real workflows

## Discussion Summary

This approach emerged from recognizing that Claude Code handles individual GCP services well but struggles with the orchestration between them. By providing connection templates rather than trying to abstract entire services, we can give Claude Code the specific knowledge it needs while maintaining flexibility.

The key insight: It's not about making perfect tools, it's about bridging the knowledge gaps where Claude Code gets stuck.