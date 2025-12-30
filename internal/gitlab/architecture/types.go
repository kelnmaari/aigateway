// Package architecture provides automatic architecture diagram generation for code.
package architecture

import "time"

// DiagramType represents the type of architecture diagram.
type DiagramType string

const (
	DiagramModuleDependency DiagramType = "module_dependency"
	DiagramCallGraph        DiagramType = "call_graph"
	DiagramDataFlow         DiagramType = "data_flow"
	DiagramPackageStructure DiagramType = "package_structure"
)

// OutputFormat represents the output format for diagrams.
type OutputFormat string

const (
	FormatMermaid OutputFormat = "mermaid"
	FormatSVG     OutputFormat = "svg"
	FormatPNG     OutputFormat = "png"
	FormatD3JSON  OutputFormat = "d3_json"
)

// Node represents a node in the architecture graph.
type Node struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`    // "package", "module", "function", "class", "file"
	FilePath    string            `json:"file_path,omitempty"`
	Language    string            `json:"language,omitempty"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Edge represents an edge (relationship) between nodes.
type Edge struct {
	Source      string            `json:"source"`
	Target      string            `json:"target"`
	Label       string            `json:"label,omitempty"`       // "imports", "calls", "uses", "extends"
	Type        string            `json:"type"`                  // "dependency", "call", "inheritance", "composition"
	Weight      int               `json:"weight,omitempty"`      // Number of interactions
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Graph represents the complete architecture graph.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Diagram represents a generated architecture diagram.
type Diagram struct {
	ID          string       `json:"id"`
	ProjectID   string       `json:"project_id"`
	Type        DiagramType  `json:"type"`
	Format      OutputFormat `json:"format"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Content     string       `json:"content"`     // Mermaid code or SVG/PNG data
	Graph       *Graph       `json:"graph,omitempty"` // Raw graph data for D3.js
	CreatedAt   time.Time    `json:"created_at"`
	Duration    string       `json:"duration"`
	TokensUsed  int          `json:"tokens_used"`
	ModelID     string       `json:"model_id,omitempty"`
}

// GenerateRequest contains parameters for diagram generation.
type GenerateRequest struct {
	ProjectID      string       `json:"project_id"`
	CollectionName string       `json:"collection_name"`
	Type           DiagramType  `json:"type"`
	Format         OutputFormat `json:"format,omitempty"`  // Default: mermaid
	ModelID        string       `json:"model_id,omitempty"`
	Scope          string       `json:"scope,omitempty"`   // Filter by path prefix
	MaxDepth       int          `json:"max_depth,omitempty"` // Max relationship depth
	Language       string       `json:"language,omitempty"` // Filter by language
}

// GenerateResult contains the result of diagram generation.
type GenerateResult struct {
	ProjectID   string    `json:"project_id"`
	GeneratedAt time.Time `json:"generated_at"`
	Duration    string    `json:"duration"`
	Status      string    `json:"status"` // "completed", "failed"
	Error       string    `json:"error,omitempty"`
	Diagrams    []Diagram `json:"diagrams"`
	TokensUsed  int       `json:"tokens_used"`
	ModelID     string    `json:"model_id"`
}

// ScanRequest contains parameters for architecture scan.
type ScanRequest struct {
	ProjectID      string `json:"project_id"`
	CollectionName string `json:"collection_name"`
	Language       string `json:"language,omitempty"`
	MaxFiles       int    `json:"max_files,omitempty"`
}

// ScanResult contains the result of architecture scan.
type ScanResult struct {
	ProjectID   string        `json:"project_id"`
	ScanID      string        `json:"scan_id"`
	ScannedAt   time.Time     `json:"scanned_at"`
	Duration    string        `json:"duration"`
	Status      string        `json:"status"`
	Error       string        `json:"error,omitempty"`
	Graph       Graph         `json:"graph"`
	Summary     ScanSummary   `json:"summary"`
	Languages   []string      `json:"languages"`
}

// ScanSummary provides overview statistics.
type ScanSummary struct {
	TotalNodes       int            `json:"total_nodes"`
	TotalEdges       int            `json:"total_edges"`
	ByNodeType       map[string]int `json:"by_node_type"`
	ByEdgeType       map[string]int `json:"by_edge_type"`
	ByLanguage       map[string]int `json:"by_language"`
	ModuleCount      int            `json:"module_count"`
	FunctionCount    int            `json:"function_count"`
	PackageCount     int            `json:"package_count"`
	CircularDeps     [][]string     `json:"circular_deps,omitempty"` // Detected circular dependencies
}

// MermaidOptions contains options for Mermaid diagram generation.
type MermaidOptions struct {
	Direction    string `json:"direction,omitempty"`    // "TB", "BT", "LR", "RL"
	Theme        string `json:"theme,omitempty"`        // "default", "dark", "forest", "neutral"
	ShowLabels   bool   `json:"show_labels,omitempty"`
	MaxNodes     int    `json:"max_nodes,omitempty"`
	GroupByType  bool   `json:"group_by_type,omitempty"`
}

