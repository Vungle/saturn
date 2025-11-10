package rag

// QueryEnhancementPromptTemplate is the template for query enhancement.
// Use {today} and {query} as placeholders that will be replaced at runtime.
const QueryEnhancementPromptTemplate = `**CONTEXT**: Today's date is {today}. Use this to understand relative date references in the query.

Analyze this business query and provide:

1. enhanced_query: Improve for semantic search by expanding abbreviations and adding relevant terms. KEEP the original query terms and ADD date context when time is mentioned.

**DATE ENRICHMENT EXAMPLES (only add when dates/time mentioned):**
- "last month" → add "September 2025"
- "last week" → add "2025-10-07 to 2025-10-13"
- "this week" → add "2025-10-14 to 2025-10-20"
- "Q3" → add "third quarter July to September 2025"
- "yesterday" → add "2025-10-16"

2. metadata_filters: Extract ONLY filters that are relevant to the query.
   IMPORTANT: If a metadata field is not referenced in the query, do not include it.

DOCUMENT-LEVEL METADATA:

**business_units:** Which Liftoff business unit(s) are relevant? (array format)
- "Demand": Helps advertisers acquire high-quality users through UA and retargeting
- "Monetize": Helps app publishers maximize revenue through in-app advertising
- "VX": Vungle Exchange - programmatic advertising platform (includes AoVX, AdColony)
- "Other": Not specific to above units
- Return as array: ["VX", "Demand"] or single: ["VX"]

**regions:** Which geographic regions are covered? (array format)
- "AMERICAS", "EMEA", "APAC"
- Return as array: ["AMERICAS", "APAC"] or single: ["APAC"]
- Use empty array [] if no specific regions mentioned

**dates:** When were the relevant reports/documents generated? (array of dates in YYYY-MM-DD format)
- Return a list of specific dates to search for reports generated on those dates
- This is when the content were created, not the data period they cover
- Format: ["YYYY-MM-DD", "YYYY-MM-DD", ...] (e.g., ["2025-10-15", "2025-10-16"])
- **IMPORTANT: You decide which specific dates to include based on semantic understanding**

**labels:** What general semantic labels describe this content? (array format)
- Financial: revenue, margins, costs, budget
- Performance: performance, conversion, volume
- Time Periods: qtd, weekly, monthly, daily, forecast
- Analysis Types: summary, trends, comparison, insights, breakdown
- Return as array: ["revenue", "qtd", "summary"]
- Be selective - don't over-label, only include clearly applicable labels

EXTRACTION GUIDELINES:
- For business_units: Only include if specific business unit/platform/product mentioned
- For labels: Only include if query clearly references these categories
- For dates: **Return a list of dates ONLY when temporal filtering is needed, otherwise return empty array []**

**WHEN TO RETURN dates for dates:**
✓ Explicit time mentions: "yesterday", "last week", "Q3 2024", "October", "2025-10-15"
✓ Recency indicators: "recent", "latest", "current", "new", "updated"
✓ Status queries: "current status", "where are we now", "latest version"
✓ Trend/progression: "growth", "trending", "improving", "declining", "changes over time"
✓ Temporal comparisons: "compared to last", "since", "after", "before"

**WHEN TO RETURN empty array [] for dates:**
✗ Knowledge/definition queries: "what is", "how to", "explain", "definition of", "process for"
✗ Person/entity focused: "John's projects", "sales team updates", "what did [person] do"
✗ Comprehensive queries: "all", "complete history", "everything about"
✗ Policy/guideline queries: "guidelines", "policies", "rules", "procedures"
✗ No temporal context: "project details", "customer information", "product features"

**DATE SELECTION EXAMPLES:**
- "data as of Nov 10" → dates: ["2025-11-10"] (exact date)
- "week of Nov 10" → dates: ["2025-11-10", "2025-11-11", "2025-11-12", "2025-11-13", "2025-11-14", "2025-11-15", "2025-11-16"] (all 7 days)
- "reports for Nov 10th and 17th" → dates: ["2025-11-10", "2025-11-17"] (specific dates)
- "Q3 monthly reports" → dates: ["2025-07-01", "2025-08-01", "2025-09-01"] (first of each month)
- "yesterday" → dates: ["{calculate yesterday}"] (one day)
- "last 3 days" → dates: ["{today}", "{today-1}", "{today-2}"] (three specific dates)
- "what is ROAS" → dates: [] (knowledge query)
- "explain NB pacing" → dates: [] (definition query)
- "John's responsibilities" → dates: [] (not time-specific)

Query: {query}

Return JSON with "enhanced_query" and "metadata_filters" fields. Only include metadata fields that apply to the query.`
