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

**generated_date:** When was this report/document generated? (string in YYYY-MM-DD format)
- Look for dates in headers, footers, titles, or metadata
- This is when the report was created, not the data period it covers
- Format: YYYY-MM-DD (e.g., "2025-10-15")

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
- For generated_date: **Return a date string ONLY when temporal filtering is needed, otherwise return null**

**WHEN TO RETURN a date for generated_date:**
✓ Explicit time mentions: "yesterday", "last week", "Q3 2024", "October", "2025-10-15"
✓ Recency indicators: "recent", "latest", "current", "new", "updated"
✓ Status queries: "current status", "where are we now", "latest version"
✓ Trend/progression: "growth", "trending", "improving", "declining", "changes over time"
✓ Temporal comparisons: "compared to last", "since", "after", "before"

**WHEN TO RETURN null for generated_date:**
✗ Knowledge/definition queries: "what is", "how to", "explain", "definition of", "process for"
✗ Person/entity focused: "John's projects", "sales team updates", "what did [person] do"
✗ Comprehensive queries: "all", "complete history", "everything about"
✗ Policy/guideline queries: "guidelines", "policies", "rules", "procedures"
✗ No temporal context: "project details", "customer information", "product features"

**Examples:**
- "recent sales performance" → generated_date: "{today}"
- "Q3 revenue" → generated_date: "2025-07-01" (calculated Q3 start)
- "how has quality improved" → generated_date: "2025-07-01" (last 3-6 months)
- "what is ROAS" → generated_date: null (knowledge query)
- "explain NB pacing" → generated_date: null (definition query)
- "what did the sales manager update" → generated_date: null (wants all updates)
- "John's responsibilities" → generated_date: null (not time-specific)
- "company policies on refunds" → generated_date: null (policy query)

Query: {query}

Return JSON with "enhanced_query" and "metadata_filters" fields. Only include metadata fields that apply to the query.`

// MetadataFieldDefinitions provides documentation for metadata fields
const MetadataFieldDefinitions = `
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

**generated_date:** When was this report/document generated? (string in YYYY-MM-DD format)
- Look for dates in headers, footers, titles, or metadata
- This is when the report was created, not the data period it covers
- Format: YYYY-MM-DD (e.g., "2025-10-15")

**labels:** What general semantic labels describe this content? (array format)
- Financial: revenue, margins, costs, budget
- Performance: performance, conversion, volume
- Time Periods: qtd, weekly, monthly, daily, forecast
- Analysis Types: summary, trends, comparison, insights, breakdown
- Return as array: ["revenue", "qtd", "summary"]
- Be selective - don't over-label, only include clearly applicable labels
`
