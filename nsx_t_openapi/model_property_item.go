/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Represents a label-value pair.
type PropertyItem struct {
	// If the condition is met then the property will be displayed. Examples of expression syntax are provided under 'example_request' section of 'CreateWidgetConfiguration' API.
	Condition string `json:"condition,omitempty"`
	// Id of drilldown widget, if any. Id should be a valid id of an existing widget. A widget is considered as drilldown widget when it is associated with any other widget and provides more detailed information about any data item from the parent widget.
	DrilldownId string `json:"drilldown_id,omitempty"`
	// Represents field value of the property.
	Field string `json:"field"`
	// Set to true if the field is a heading. Default is false.
	Heading bool `json:"heading,omitempty"`
	Label *Label `json:"label,omitempty"`
	// Label value separator used between label and value. It can be any separator like \":\"  or \"-\".
	LabelValueSeparator string `json:"label_value_separator,omitempty"`
	// Hyperlink of the specified UI page that provides details. This will be linked with value of the property.
	Navigation string `json:"navigation,omitempty"`
	// Render configuration to be applied, if any.
	RenderConfiguration []RenderConfiguration `json:"render_configuration,omitempty"`
	// Represent the vertical span of the widget / container
	Rowspan int32 `json:"rowspan,omitempty"`
	// If true, separates this property in a widget.
	Separator bool `json:"separator,omitempty"`
	// Represent the horizontal span of the widget / container.
	Span int32 `json:"span,omitempty"`
	// A style object applicable for the property item. It could be the any padding, margin style sheet applicable to the property item. A 'style' property is supported in case of layout 'AUTO' only.
	Style interface{} `json:"style,omitempty"`
	// Data type of the field.
	Type_ string `json:"type"`
}
