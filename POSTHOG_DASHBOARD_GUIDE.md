# PostHog Dashboard Management Guide

## 🔧 Setup & Authentication

### API Configuration
- **PostHog Instance**: `https://eu.posthog.com`
- **Project ID**: `83769`
- **API Key**: `phx_F67d2qCmEbroaXJ7i286kZzzxilYYjhiHVY7sxHnYDpMzvV`

### Base API Commands
```bash
# Base URL for all API calls
BASE_URL="https://eu.posthog.com/api/projects/83769"
AUTH_HEADER="Authorization: Bearer phx_F67d2qCmEbroaXJ7i286kZzzxilYYjhiHVY7sxHnYDpMzvV"

# Test API connection
curl -H "$AUTH_HEADER" "$BASE_URL/insights/"
```

## 📊 Current Dashboard Structure

### Dashboard IDs
- **Mix - Tool Performance**: `233620`
- **Overview Dashboard**: `233638` 
- **Mix - Conversation Analytics**: `233649`

### Event Types Tracked
- `tool_call` - Tool executions with properties: `tool_name`, `success`, `error`, `session_id`
- `user_message` - User inputs with properties: `session_id`, `message_id`, `content`, `model`
- `agent_response` - AI responses with properties: `session_id`, `message_id`, `content`, `model`

## 🛠 Common Operations

### 1. List All Dashboards
```bash
curl -H "$AUTH_HEADER" "$BASE_URL/dashboards/"
```

### 2. List All Insights
```bash
curl -H "$AUTH_HEADER" "$BASE_URL/insights/"
```

### 3. Create New Dashboard
```bash
curl -X POST -H "$AUTH_HEADER" -H "Content-Type: application/json" -d '{
  "name": "Dashboard Name",
  "description": "Dashboard description"
}' "$BASE_URL/dashboards/"
```

### 4. Create New Insight
```bash
curl -X POST -H "$AUTH_HEADER" -H "Content-Type: application/json" -d '{
  "name": "Insight Name",
  "query": { QUERY_OBJECT },
  "dashboards": [DASHBOARD_ID]
}' "$BASE_URL/insights/"
```

## 📈 Insight Templates

### Tool Success Rate (Number)
```json
{
  "name": "Tool Success Rate %",
  "query": {
    "kind": "InsightVizNode",
    "source": {
      "kind": "TrendsQuery",
      "series": [
        {
          "kind": "EventsNode",
          "event": "tool_call",
          "math": "total",
          "properties": [{"key": "success", "type": "event", "value": "true", "operator": "exact"}],
          "name": "Successful calls"
        },
        {
          "kind": "EventsNode", 
          "event": "tool_call",
          "math": "total",
          "name": "Total calls"
        }
      ],
      "trendsFilter": {
        "display": "BoldNumber",
        "formulaNodes": [{"formula": "A / B * 100"}]
      },
      "dateRange": {"date_from": "-7d", "date_to": null}
    }
  },
  "dashboards": [233620]
}
```

### Tool Usage Breakdown (Bar Chart)
```json
{
  "name": "Tool Usage by Type",
  "query": {
    "kind": "InsightVizNode",
    "source": {
      "kind": "TrendsQuery",
      "series": [{
        "kind": "EventsNode",
        "event": "tool_call",
        "math": "total"
      }],
      "breakdownFilter": {
        "breakdowns": [{"type": "event", "property": "tool_name"}]
      },
      "trendsFilter": {"display": "ActionsBar"},
      "dateRange": {"date_from": "-7d", "date_to": null}
    }
  },
  "dashboards": [233620]
}
```

### Daily Active Sessions (Line Chart)
```json
{
  "name": "Daily Active Sessions",
  "query": {
    "kind": "InsightVizNode",
    "source": {
      "kind": "TrendsQuery",
      "series": [{
        "kind": "EventsNode",
        "event": "user_message",
        "math": "dau"
      }],
      "interval": "day",
      "dateRange": {"date_from": "-30d", "date_to": null},
      "trendsFilter": {"display": "ActionsLineGraph"}
    }
  },
  "dashboards": [233638]
}
```

### Failed Tool Calls (Table)
```json
{
  "name": "Failed Tool Calls",
  "query": {
    "kind": "InsightVizNode",
    "source": {
      "kind": "TrendsQuery",
      "series": [{
        "kind": "EventsNode",
        "event": "tool_call",
        "math": "total",
        "properties": [{"key": "success", "type": "event", "value": "false", "operator": "exact"}]
      }],
      "breakdownFilter": {
        "breakdowns": [
          {"type": "event", "property": "tool_name"},
          {"type": "event", "property": "error"}
        ]
      },
      "trendsFilter": {"display": "ActionsTable"}
    }
  },
  "dashboards": [233620]
}
```

### Model Usage Distribution (Pie Chart)
```json
{
  "name": "Model Usage Distribution",
  "query": {
    "kind": "InsightVizNode",
    "source": {
      "kind": "TrendsQuery",
      "series": [{
        "kind": "EventsNode",
        "event": "agent_response",
        "math": "total"
      }],
      "breakdownFilter": {
        "breakdowns": [{"type": "event", "property": "model"}]
      },
      "trendsFilter": {"display": "ActionsPie"},
      "dateRange": {"date_from": "-7d", "date_to": null}
    }
  },
  "dashboards": [233638]
}
```

## 🔍 Current Analytics Summary

### Key Metrics (as of Aug 19, 2025)
- **Total Tool Calls**: 324 (last 7 days)
- **Tool Success Rate**: 65.4%
- **Most Used Tools**: 
  - bash: 162 calls (50%)
  - todo_write: 52 calls (16%)
  - media_showcase: 33 calls (10%)
  - ls: 30 calls (9%)

### Activity Patterns
- **Aug 18**: 66 tool calls
- **Aug 19**: 258 tool calls (390% increase)
- **Recent spike indicates high user engagement**

## 🚨 Alerts & Monitoring

### Recommended Alerts
1. **Tool Success Rate < 80%** - indicates system issues
2. **Zero Tool Calls for 2+ hours** - system down
3. **Bash Failure Rate > 10%** - command execution problems
4. **Daily Sessions Drop > 50%** - user engagement issue

### Creating Alerts (Manual)
1. Go to PostHog insight
2. Click "Set up alert"
3. Configure threshold and notification channel

## 📝 Quick Scripts

### Get All Tool Names
```bash
curl -H "$AUTH_HEADER" "$BASE_URL/events/?event=tool_call" | jq '.results[].properties.tool_name' | sort | uniq
```

### Get Recent Failed Tools
```bash
curl -H "$AUTH_HEADER" "$BASE_URL/events/?event=tool_call&properties=%7B%22success%22%3A%22false%22%7D" | jq '.results[] | {tool: .properties.tool_name, error: .properties.error, time: .timestamp}'
```

### Create Multiple Insights Script
```bash
#!/bin/bash
BASE_URL="https://eu.posthog.com/api/projects/83769"
AUTH_HEADER="Authorization: Bearer phx_F67d2qCmEbroaXJ7i286kZzzxilYYjhiHVY7sxHnYDpMzvV"

# Create insight function
create_insight() {
  local name="$1"
  local query="$2" 
  local dashboard_id="$3"
  
  curl -X POST -H "$AUTH_HEADER" -H "Content-Type: application/json" -d "{
    \"name\": \"$name\",
    \"query\": $query,
    \"dashboards\": [$dashboard_id]
  }" "$BASE_URL/insights/"
}

# Usage example:
# create_insight "New Metric" '{"kind":"InsightVizNode",...}' 233620
```

## 🎯 Future Enhancements

### Additional Insights to Consider
1. **Response Time Tracking** - add duration to tool_call events
2. **Error Categories** - categorize errors by type
3. **User Journey Analysis** - track session flow patterns
4. **Tool Adoption Rate** - track first-time tool usage
5. **Peak Usage Hours** - identify busy periods

### Advanced Features
1. **Cohort Analysis** - track user retention
2. **Funnel Analysis** - conversion from message to tool usage  
3. **Feature Flags** - A/B test different tool implementations
4. **Custom Events** - track specific business metrics

## 🔗 Useful Links
- **PostHog Dashboard**: https://eu.posthog.com/project/83769/dashboard/
- **API Documentation**: https://posthog.com/docs/api
- **Query Format Guide**: https://posthog.com/docs/hogql

## 💡 Tips
1. **Always test queries** in PostHog UI before automating
2. **Use consistent naming** for easy dashboard organization  
3. **Set up alerts early** to catch issues quickly
4. **Regular review** - weekly dashboard review recommended
5. **Backup important insights** - export JSON configurations

---

*Last Updated: August 19, 2025*
*Tool Success Rate: 65.4% (target: 85%+)*