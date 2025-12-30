// Package architecture provides automatic architecture diagram generation.
package architecture

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aigateway/internal/rag/vector"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Generator generates architecture diagrams from code analysis.
type Generator struct {
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewGenerator creates a new architecture diagram generator.
func NewGenerator(vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *Generator {
	return &Generator{
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// Scan analyzes the codebase and builds the architecture graph.
func (g *Generator) Scan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	startTime := time.Now()
	result := &ScanResult{
		ProjectID: req.ProjectID,
		ScanID:    uuid.New().String(),
		ScannedAt: startTime,
		Status:    "completed",
		Graph:     Graph{Nodes: []Node{}, Edges: []Edge{}},
	}

	g.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
	}).Info("Starting architecture scan")

	// Fetch code chunks from vector store
	chunks, err := g.collectChunks(ctx, req.CollectionName)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	if len(chunks) == 0 {
		result.Status = "completed"
		result.Summary = ScanSummary{}
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	// Build graph from chunks
	graph, languages := g.buildGraph(chunks)
	result.Graph = graph
	result.Languages = languages

	// Build summary
	result.Summary = g.buildSummary(graph)
	result.Duration = time.Since(startTime).String()

	g.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"nodes":      len(graph.Nodes),
		"edges":      len(graph.Edges),
		"duration":   result.Duration,
	}).Info("Architecture scan completed")

	return result, nil
}

// Generate creates architecture diagrams from the scan result.
func (g *Generator) Generate(ctx context.Context, req GenerateRequest, scanResult *ScanResult) (*GenerateResult, error) {
	startTime := time.Now()
	result := &GenerateResult{
		ProjectID:   req.ProjectID,
		GeneratedAt: startTime,
		Status:      "completed",
		Diagrams:    []Diagram{},
		ModelID:     req.ModelID,
	}

	g.logger.WithFields(logrus.Fields{
		"project_id":   req.ProjectID,
		"diagram_type": req.Type,
		"format":       req.Format,
	}).Info("Starting diagram generation")

	// Default format is Mermaid
	format := req.Format
	if format == "" {
		format = FormatMermaid
	}

	var diagram Diagram
	var err error

	switch req.Type {
	case DiagramModuleDependency:
		diagram, err = g.generateModuleDependencyDiagram(ctx, req, scanResult)
	case DiagramCallGraph:
		diagram, err = g.generateCallGraphDiagram(ctx, req, scanResult)
	case DiagramPackageStructure:
		diagram, err = g.generatePackageStructureDiagram(ctx, req, scanResult)
	case DiagramDataFlow:
		diagram, err = g.generateDataFlowDiagram(ctx, req, scanResult)
	default:
		// Generate all diagram types
		for _, diagramType := range []DiagramType{DiagramModuleDependency, DiagramPackageStructure} {
			reqCopy := req
			reqCopy.Type = diagramType
			d, e := g.generateDiagramByType(ctx, reqCopy, scanResult)
			if e == nil {
				result.Diagrams = append(result.Diagrams, d)
				result.TokensUsed += d.TokensUsed
			}
		}
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	result.Diagrams = append(result.Diagrams, diagram)
	result.TokensUsed = diagram.TokensUsed
	result.Duration = time.Since(startTime).String()

	return result, nil
}

func (g *Generator) generateDiagramByType(ctx context.Context, req GenerateRequest, scanResult *ScanResult) (Diagram, error) {
	switch req.Type {
	case DiagramModuleDependency:
		return g.generateModuleDependencyDiagram(ctx, req, scanResult)
	case DiagramCallGraph:
		return g.generateCallGraphDiagram(ctx, req, scanResult)
	case DiagramPackageStructure:
		return g.generatePackageStructureDiagram(ctx, req, scanResult)
	case DiagramDataFlow:
		return g.generateDataFlowDiagram(ctx, req, scanResult)
	default:
		return g.generateModuleDependencyDiagram(ctx, req, scanResult)
	}
}

func (g *Generator) generateModuleDependencyDiagram(ctx context.Context, req GenerateRequest, scanResult *ScanResult) (Diagram, error) {
	diagram := Diagram{
		ID:        uuid.New().String(),
		ProjectID: req.ProjectID,
		Type:      DiagramModuleDependency,
		Format:    FormatMermaid,
		Title:     "Module Dependency Graph",
		CreatedAt: time.Now(),
	}

	// Filter nodes to packages/modules only
	var moduleNodes []Node
	moduleEdges := []Edge{}

	for _, node := range scanResult.Graph.Nodes {
		if node.Type == "package" || node.Type == "module" || node.Type == "file" {
			moduleNodes = append(moduleNodes, node)
		}
	}

	for _, edge := range scanResult.Graph.Edges {
		if edge.Type == "dependency" || edge.Label == "imports" {
			moduleEdges = append(moduleEdges, edge)
		}
	}

	// Generate Mermaid diagram
	var mermaid strings.Builder
	mermaid.WriteString("graph TD\n")

	// Add nodes
	for _, node := range moduleNodes {
		nodeName := sanitizeMermaidID(node.ID)
		label := node.Name
		if len(label) > 30 {
			label = label[:27] + "..."
		}
		mermaid.WriteString(fmt.Sprintf("    %s[\"%s\"]\n", nodeName, label))
	}

	// Add edges
	for _, edge := range moduleEdges {
		source := sanitizeMermaidID(edge.Source)
		target := sanitizeMermaidID(edge.Target)
		if edge.Label != "" {
			mermaid.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", source, edge.Label, target))
		} else {
			mermaid.WriteString(fmt.Sprintf("    %s --> %s\n", source, target))
		}
	}

	diagram.Content = mermaid.String()
	diagram.Graph = &Graph{Nodes: moduleNodes, Edges: moduleEdges}
	diagram.Duration = "0s"

	return diagram, nil
}

func (g *Generator) generateCallGraphDiagram(ctx context.Context, req GenerateRequest, scanResult *ScanResult) (Diagram, error) {
	diagram := Diagram{
		ID:        uuid.New().String(),
		ProjectID: req.ProjectID,
		Type:      DiagramCallGraph,
		Format:    FormatMermaid,
		Title:     "Function Call Graph",
		CreatedAt: time.Now(),
	}

	// Filter nodes to functions only
	var funcNodes []Node
	funcEdges := []Edge{}

	for _, node := range scanResult.Graph.Nodes {
		if node.Type == "function" || node.Type == "method" {
			funcNodes = append(funcNodes, node)
		}
	}

	for _, edge := range scanResult.Graph.Edges {
		if edge.Type == "call" || edge.Label == "calls" {
			funcEdges = append(funcEdges, edge)
		}
	}

	// Limit nodes for readability
	maxNodes := 50
	if len(funcNodes) > maxNodes {
		funcNodes = funcNodes[:maxNodes]
	}

	// Generate Mermaid diagram
	var mermaid strings.Builder
	mermaid.WriteString("graph LR\n")

	for _, node := range funcNodes {
		nodeName := sanitizeMermaidID(node.ID)
		label := node.Name
		if len(label) > 25 {
			label = label[:22] + "..."
		}
		mermaid.WriteString(fmt.Sprintf("    %s((\"%s\"))\n", nodeName, label))
	}

	for _, edge := range funcEdges {
		source := sanitizeMermaidID(edge.Source)
		target := sanitizeMermaidID(edge.Target)
		mermaid.WriteString(fmt.Sprintf("    %s --> %s\n", source, target))
	}

	diagram.Content = mermaid.String()
	diagram.Graph = &Graph{Nodes: funcNodes, Edges: funcEdges}
	diagram.Duration = "0s"

	return diagram, nil
}

func (g *Generator) generatePackageStructureDiagram(ctx context.Context, req GenerateRequest, scanResult *ScanResult) (Diagram, error) {
	diagram := Diagram{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		Type:        DiagramPackageStructure,
		Format:      FormatMermaid,
		Title:       "Package Structure",
		Description: "Hierarchical view of project packages and modules",
		CreatedAt:   time.Now(),
	}

	// Group nodes by package/directory
	packages := make(map[string][]Node)
	for _, node := range scanResult.Graph.Nodes {
		pkgPath := extractPackagePath(node.FilePath)
		packages[pkgPath] = append(packages[pkgPath], node)
	}

	// Generate Mermaid flowchart with subgraphs
	var mermaid strings.Builder
	mermaid.WriteString("graph TB\n")

	for pkgPath, nodes := range packages {
		pkgID := sanitizeMermaidID(pkgPath)
		pkgName := pkgPath
		if pkgName == "" {
			pkgName = "root"
		}

		mermaid.WriteString(fmt.Sprintf("    subgraph %s[\"%s\"]\n", pkgID, pkgName))
		for _, node := range nodes {
			nodeID := sanitizeMermaidID(node.ID)
			label := node.Name
			switch node.Type {
			case "function", "method":
				mermaid.WriteString(fmt.Sprintf("        %s((\"%s\"))\n", nodeID, label))
			case "class", "type":
				mermaid.WriteString(fmt.Sprintf("        %s{{\"%s\"}}\n", nodeID, label))
			default:
				mermaid.WriteString(fmt.Sprintf("        %s[\"%s\"]\n", nodeID, label))
			}
		}
		mermaid.WriteString("    end\n")
	}

	diagram.Content = mermaid.String()
	diagram.Duration = "0s"

	return diagram, nil
}

func (g *Generator) generateDataFlowDiagram(ctx context.Context, req GenerateRequest, scanResult *ScanResult) (Diagram, error) {
	diagram := Diagram{
		ID:          uuid.New().String(),
		ProjectID:   req.ProjectID,
		Type:        DiagramDataFlow,
		Format:      FormatMermaid,
		Title:       "Data Flow Diagram",
		Description: "Shows how data flows through the system",
		CreatedAt:   time.Now(),
	}

	// Use LLM to analyze data flow
	if req.ModelID != "" {
		content, tokens, err := g.generateWithLLM(ctx, req.ModelID, scanResult)
		if err == nil {
			diagram.Content = content
			diagram.TokensUsed = tokens
			diagram.Duration = "0s"
			return diagram, nil
		}
		g.logger.WithError(err).Warn("LLM data flow generation failed, using fallback")
	}

	// Fallback: basic data flow from edges
	var mermaid strings.Builder
	mermaid.WriteString("graph LR\n")
	mermaid.WriteString("    subgraph Input\n")
	mermaid.WriteString("        A[User Input]\n")
	mermaid.WriteString("    end\n")
	mermaid.WriteString("    subgraph Processing\n")
	mermaid.WriteString("        B[Handler]\n")
	mermaid.WriteString("        C[Service]\n")
	mermaid.WriteString("    end\n")
	mermaid.WriteString("    subgraph Output\n")
	mermaid.WriteString("        D[Database]\n")
	mermaid.WriteString("        E[Response]\n")
	mermaid.WriteString("    end\n")
	mermaid.WriteString("    A --> B\n")
	mermaid.WriteString("    B --> C\n")
	mermaid.WriteString("    C --> D\n")
	mermaid.WriteString("    C --> E\n")

	diagram.Content = mermaid.String()
	diagram.Duration = "0s"

	return diagram, nil
}

func (g *Generator) generateWithLLM(ctx context.Context, modelID string, scanResult *ScanResult) (string, int, error) {
	// Build context from scan result
	var sb strings.Builder
	sb.WriteString("Project structure:\n")
	for _, node := range scanResult.Graph.Nodes[:min(100, len(scanResult.Graph.Nodes))] {
		sb.WriteString(fmt.Sprintf("- %s (%s) in %s\n", node.Name, node.Type, node.FilePath))
	}
	sb.WriteString("\nDependencies:\n")
	for _, edge := range scanResult.Graph.Edges[:min(50, len(scanResult.Graph.Edges))] {
		sb.WriteString(fmt.Sprintf("- %s -> %s (%s)\n", edge.Source, edge.Target, edge.Label))
	}

	prompt := fmt.Sprintf(`Analyze this codebase structure and generate a Mermaid data flow diagram.
Show how data flows from input (API endpoints, user input) through services to storage and output.

%s

Generate ONLY the Mermaid diagram code (starting with "graph LR" or "graph TD"), no explanation.
Use subgraphs to group related components. Make it readable and informative.`, sb.String())

	response, tokens, err := g.callLLM(ctx, modelID, prompt)
	if err != nil {
		return "", tokens, err
	}

	// Extract Mermaid code
	mermaid := extractMermaidCode(response)
	return mermaid, tokens, nil
}

func (g *Generator) collectChunks(ctx context.Context, collectionName string) ([]codeChunk, error) {
	if g.vectorStore == nil {
		return nil, fmt.Errorf("vector store not configured")
	}

	var chunks []codeChunk
	seen := make(map[string]bool)
	maxChunks := 500

	// Use ScrollAll to retrieve all code chunks
	err := g.vectorStore.ScrollAll(ctx, collectionName, nil, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			if len(chunks) >= maxChunks {
				return nil
			}

			filePath, _ := doc.Metadata["file_path"].(string)
			if filePath == "" || seen[filePath] {
				continue
			}
			seen[filePath] = true

			content := doc.Text
			if c, ok := doc.Metadata["content"].(string); ok && c != "" {
				content = c
			}

			chunks = append(chunks, codeChunk{
				FilePath: filePath,
				Content:  content,
				Language: detectLanguage(filePath),
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scroll failed: %w", err)
	}

	return chunks, nil
}

func (g *Generator) buildGraph(chunks []codeChunk) (Graph, []string) {
	graph := Graph{
		Nodes: []Node{},
		Edges: []Edge{},
	}
	languagesMap := make(map[string]bool)
	nodeMap := make(map[string]bool)

	for _, chunk := range chunks {
		languagesMap[chunk.Language] = true

		// Create file node
		fileNode := Node{
			ID:       chunk.FilePath,
			Name:     extractFileName(chunk.FilePath),
			Type:     "file",
			FilePath: chunk.FilePath,
			Language: chunk.Language,
		}
		if !nodeMap[fileNode.ID] {
			graph.Nodes = append(graph.Nodes, fileNode)
			nodeMap[fileNode.ID] = true
		}

		// Extract imports and create dependency edges
		imports := extractImports(chunk.Content, chunk.Language)
		for _, imp := range imports {
			edge := Edge{
				Source: chunk.FilePath,
				Target: imp,
				Label:  "imports",
				Type:   "dependency",
			}
			graph.Edges = append(graph.Edges, edge)

			// Create target node if not exists
			if !nodeMap[imp] {
				graph.Nodes = append(graph.Nodes, Node{
					ID:   imp,
					Name: extractFileName(imp),
					Type: "module",
				})
				nodeMap[imp] = true
			}
		}

		// Extract functions
		functions := extractFunctions(chunk.Content, chunk.Language)
		for _, fn := range functions {
			fnNode := Node{
				ID:       chunk.FilePath + "#" + fn,
				Name:     fn,
				Type:     "function",
				FilePath: chunk.FilePath,
				Language: chunk.Language,
			}
			if !nodeMap[fnNode.ID] {
				graph.Nodes = append(graph.Nodes, fnNode)
				nodeMap[fnNode.ID] = true
			}
		}
	}

	var languages []string
	for lang := range languagesMap {
		languages = append(languages, lang)
	}

	return graph, languages
}

func (g *Generator) buildSummary(graph Graph) ScanSummary {
	summary := ScanSummary{
		TotalNodes: len(graph.Nodes),
		TotalEdges: len(graph.Edges),
		ByNodeType: make(map[string]int),
		ByEdgeType: make(map[string]int),
		ByLanguage: make(map[string]int),
	}

	for _, node := range graph.Nodes {
		summary.ByNodeType[node.Type]++
		if node.Language != "" {
			summary.ByLanguage[node.Language]++
		}
		switch node.Type {
		case "module", "package":
			summary.ModuleCount++
		case "function", "method":
			summary.FunctionCount++
		}
	}

	for _, edge := range graph.Edges {
		summary.ByEdgeType[edge.Type]++
	}

	summary.PackageCount = summary.ByNodeType["package"]

	return summary
}

func (g *Generator) callLLM(ctx context.Context, modelID, prompt string) (string, int, error) {
	reqBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  4000,
		"temperature": 0.3,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", g.llmBaseURL+"/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if g.llmAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+g.llmAPIKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", 0, err
	}

	if len(result.Choices) == 0 {
		return "", result.Usage.TotalTokens, fmt.Errorf("no response from LLM")
	}

	return result.Choices[0].Message.Content, result.Usage.TotalTokens, nil
}

type codeChunk struct {
	FilePath string
	Content  string
	Language string
}

// Helper functions

func sanitizeMermaidID(id string) string {
	// Replace special characters for Mermaid compatibility
	result := strings.ReplaceAll(id, "/", "_")
	result = strings.ReplaceAll(result, ".", "_")
	result = strings.ReplaceAll(result, "-", "_")
	result = strings.ReplaceAll(result, "@", "_")
	result = strings.ReplaceAll(result, "#", "_")
	result = strings.ReplaceAll(result, " ", "_")
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}

func extractPackagePath(filePath string) string {
	parts := strings.Split(filePath, "/")
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], "/")
	}
	return ""
}

func extractFileName(filePath string) string {
	parts := strings.Split(filePath, "/")
	return parts[len(parts)-1]
}

func detectLanguage(filePath string) string {
	switch {
	case strings.HasSuffix(filePath, ".go"):
		return "go"
	case strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".tsx"):
		return "typescript"
	case strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".jsx"):
		return "javascript"
	case strings.HasSuffix(filePath, ".py"):
		return "python"
	case strings.HasSuffix(filePath, ".rs"):
		return "rust"
	case strings.HasSuffix(filePath, ".java"):
		return "java"
	default:
		return "unknown"
	}
}

func extractImports(content, language string) []string {
	var imports []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch language {
		case "go":
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "\"") {
				// Extract import path
				start := strings.Index(line, "\"")
				end := strings.LastIndex(line, "\"")
				if start != -1 && end > start {
					imports = append(imports, line[start+1:end])
				}
			}
		case "typescript", "javascript":
			if strings.Contains(line, "import ") || strings.Contains(line, "require(") {
				// Extract module name
				if strings.Contains(line, "from ") {
					parts := strings.Split(line, "from ")
					if len(parts) > 1 {
						modPart := strings.Trim(parts[1], " '\";")
						imports = append(imports, modPart)
					}
				}
			}
		case "python":
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					imports = append(imports, parts[1])
				}
			}
		}
	}

	return imports
}

func extractFunctions(content, language string) []string {
	var functions []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch language {
		case "go":
			if strings.HasPrefix(line, "func ") {
				// Extract function name
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					name := parts[1]
					if idx := strings.Index(name, "("); idx > 0 {
						name = name[:idx]
					}
					functions = append(functions, name)
				}
			}
		case "typescript", "javascript":
			if strings.Contains(line, "function ") || strings.Contains(line, "const ") || strings.Contains(line, "async ") {
				// Simple function detection
				if strings.Contains(line, "(") {
					parts := strings.Fields(line)
					for i, p := range parts {
						if p == "function" && i+1 < len(parts) {
							name := parts[i+1]
							if idx := strings.Index(name, "("); idx > 0 {
								name = name[:idx]
							}
							functions = append(functions, name)
						}
					}
				}
			}
		case "python":
			if strings.HasPrefix(line, "def ") || strings.HasPrefix(line, "async def ") {
				parts := strings.Fields(line)
				for i, p := range parts {
					if p == "def" && i+1 < len(parts) {
						name := parts[i+1]
						if idx := strings.Index(name, "("); idx > 0 {
							name = name[:idx]
						}
						functions = append(functions, name)
					}
				}
			}
		}
	}

	return functions
}

func extractMermaidCode(response string) string {
	// Try to extract code block
	if idx := strings.Index(response, "```mermaid"); idx != -1 {
		start := idx + 10
		end := strings.Index(response[start:], "```")
		if end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}
	if idx := strings.Index(response, "```"); idx != -1 {
		start := idx + 3
		// Skip language tag if present
		if newline := strings.Index(response[start:], "\n"); newline != -1 {
			start = start + newline + 1
		}
		end := strings.Index(response[start:], "```")
		if end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}
	// Return as-is if no code block found
	return strings.TrimSpace(response)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

