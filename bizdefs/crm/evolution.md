# CRM Evolution History

## 2024-05-19: Schema Modernization (v2)

The CRM BizDef was updated to align with global sales standards. 

### Changes to `crm_lead`:
- **Status Renaming**: The internal field `old_status` was renamed to `status` to provide a cleaner API for external integrations.
- **Lead Scoring**: Introduced a new `lead_score` field (defaulting to 0). This field is intended to be used by the upcoming `lead_ranking_agent` to prioritize sales efforts.

### Migration Instructions:
All existing leads must be migrated to `v2` to enable lead scoring functionality. This can be triggered via the `cms_bizdef_evolve` tool.
