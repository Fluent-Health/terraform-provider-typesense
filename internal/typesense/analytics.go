package typesense

import "context"

// AnalyticsRuleParams are aggregation parameters shared by create and update.
type AnalyticsRuleParams struct {
	DestinationCollection *string   `json:"destination_collection,omitempty"`
	CounterField          *string   `json:"counter_field,omitempty"`
	Limit                 *int      `json:"limit,omitempty"`
	ExpandQuery           *bool     `json:"expand_query,omitempty"`
	Weight                *int      `json:"weight,omitempty"`
	CaptureSearchRequests *bool     `json:"capture_search_requests,omitempty"`
	MetaFields            *[]string `json:"meta_fields,omitempty"`
}

// AnalyticsRule is the server's representation of an analytics rule.
type AnalyticsRule struct {
	Name       string               `json:"name"`
	Type       string               `json:"type"`
	Collection string               `json:"collection"`
	EventType  string               `json:"event_type"`
	RuleTag    *string              `json:"rule_tag,omitempty"`
	Params     *AnalyticsRuleParams `json:"params,omitempty"`
}

// AnalyticsRuleCreate is the body sent to POST /analytics/rules (as an array).
type AnalyticsRuleCreate struct {
	Name       string               `json:"name"`
	Type       string               `json:"type"`
	Collection string               `json:"collection"`
	EventType  string               `json:"event_type"`
	RuleTag    *string              `json:"rule_tag,omitempty"`
	Params     *AnalyticsRuleParams `json:"params,omitempty"`
}

// AnalyticsRuleUpdate is the body sent to PATCH /analytics/rules/{name}.
type AnalyticsRuleUpdate struct {
	Params  *AnalyticsRuleParams `json:"params,omitempty"`
	RuleTag *string              `json:"rule_tag,omitempty"`
}

// CreateAnalyticsRules submits one or more rule creates and returns the parsed rules.
func (c *Client) CreateAnalyticsRules(ctx context.Context, rules []AnalyticsRuleCreate) ([]AnalyticsRule, error) {
	var out []AnalyticsRule
	if err := c.do(ctx, "POST", "/analytics/rules", nil, rules, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetAnalyticsRule retrieves an analytics rule by name.
func (c *Client) GetAnalyticsRule(ctx context.Context, name string) (*AnalyticsRule, error) {
	out := &AnalyticsRule{}
	if err := c.do(ctx, "GET", "/analytics/rules/"+name, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateAnalyticsRule patches an analytics rule.
func (c *Client) UpdateAnalyticsRule(ctx context.Context, name string, body *AnalyticsRuleUpdate) (*AnalyticsRule, error) {
	out := &AnalyticsRule{}
	if err := c.do(ctx, "PATCH", "/analytics/rules/"+name, nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// DeleteAnalyticsRule removes an analytics rule.
func (c *Client) DeleteAnalyticsRule(ctx context.Context, name string) error {
	return c.do(ctx, "DELETE", "/analytics/rules/"+name, nil, nil, nil)
}
