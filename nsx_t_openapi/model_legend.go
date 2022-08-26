/*
 * NSX-T Data Center Policy API
 *
 * VMware NSX-T Data Center Policy REST API
 *
 * API version: 3.2.1.0.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package nsx_t_openapi

// Represents legend that describes the entities of the widget.
type Legend struct {
	// Describes the alignment of legend. Alignment of a legend denotes how individual items of the legend are aligned in a container. For example, if VERTICAL is chosen then the items of the legend will appear one below the other and if HORIZONTAL is chosen then the items will appear side by side.
	Alignment string `json:"alignment,omitempty"`
	// If set to true, it will display the counts in legend. If set to false, counts of entities are not displayed in the legend.
	DisplayCount bool `json:"display_count,omitempty"`
	// Display mode for legends.
	DisplayMode string `json:"display_mode,omitempty"`
	// Show checkbox along with legends if value is set to true. Widget filtering capability can be enable based on legend checkbox selection. for 'display_mode' SHOW_OTHER_GROUP_WITH_LEGENDS filterable property is not supported.
	Filterable bool `json:"filterable,omitempty"`
	// A minimum number of legends to be displayed upfront. if 'display_mode' is set to SHOW_MIN_NO_OF_LEGENDS then this property value will be used to display number of legends upfront in the UI.
	MinLegendsDisplayCount int32 `json:"min_legends_display_count,omitempty"`
	// A translated label for showing other category label in legends.
	OtherGroupLegendLabel string `json:"other_group_legend_label,omitempty"`
	// Describes the relative placement of legend. The legend of a widget can be placed either to the TOP or BOTTOM or LEFT or RIGHT relative to the widget. For example, if RIGHT is chosen then legend is placed to the right of the widget.
	Position string `json:"position,omitempty"`
	// Describes the render type for the legend. The legend for an entity describes the entity in the widget. The supported legend type is a circle against which the entity's details such as display_name are shown. The color of the circle denotes the color of the entity shown inside the widget.
	Type_ string `json:"type,omitempty"`
	// Show unit of entities in the legend.
	Unit string `json:"unit,omitempty"`
}
