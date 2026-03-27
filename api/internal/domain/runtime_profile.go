package domain

type RuntimeProfileAudience struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RuntimeProfileArtifact struct {
	Label   string `json:"label"`
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
}

type RuntimeProfile struct {
	ProductName      string                   `json:"product_name"`
	Tagline          string                   `json:"tagline"`
	PrimaryUseCase   string                   `json:"primary_use_case"`
	CoreProblem      string                   `json:"core_problem"`
	ValueProposition string                   `json:"value_proposition"`
	TargetUsers      []RuntimeProfileAudience `json:"target_users"`
	CurrentScope     []string                 `json:"current_scope"`
	Differentiators  []string                 `json:"differentiators"`
	NonGoals         []string                 `json:"non_goals"`
	ResearchTracks   []string                 `json:"research_tracks"`
	ArtifactRefs     []RuntimeProfileArtifact `json:"artifact_refs"`
	LastReviewedOn   string                   `json:"last_reviewed_on"`
}

func CurrentRuntimeProfile() RuntimeProfile {
	return RuntimeProfile{
		ProductName: "ThingIndex",
		Tagline:     "Relational document intelligence for auditable operational knowledge.",
		PrimaryUseCase: "Organize collections of operational, compliance, legal, and research documents with typed relational indexing, " +
			"recoverable context, and grounded follow-up workflows.",
		CoreProblem: "Document-heavy teams often have storage, metadata, and AI answers split across disconnected tools, which makes " +
			"context recovery, provenance, and operational auditing difficult.",
		ValueProposition: "ThingIndex keeps document organization, structured extraction, context recovery, and provider portability in " +
			"a single relational runtime that can be self-hosted or consumed as an API without hiding provenance behind a vector-first stack.",
		TargetUsers: []RuntimeProfileAudience{
			{
				Name:        "Operations and compliance teams",
				Description: "Need searchable workspaces, explicit provenance, and repeatable document handling across collections.",
			},
			{
				Name:        "Legal and policy workflows",
				Description: "Benefit from collection-scoped context, structured extraction, and grounded follow-up review.",
			},
			{
				Name:        "Academic and research environments",
				Description: "Need a runtime that is inspectable, reproducible, and easy to defend technically in a TCC or article.",
			},
		},
		CurrentScope: []string{
			"JWT-authenticated document workspace with categories, collections, and uploads",
			"Structured indexing with paragraphs, tables, outline nodes, metadata pairs, and monitoring telemetry",
			"Relational collection-context recovery in PostgreSQL",
			"Grounded chat by collection with provider runtime separation for chat and indexing",
			"Self-hosted stack with Go API, Next.js frontend, PostgreSQL, MinIO, RabbitMQ, and versioned API docs",
		},
		Differentiators: []string{
			"Relational-first context reconstruction instead of vector-first navigation",
			"Explicit separation between current runtime, experimental provider paths, and research-only extensions",
			"Operational visibility with audit trails, readiness checks, monitoring, and documented support levels",
		},
		NonGoals: []string{
			"Do not present the main runtime as a generic AI copilot detached from document provenance",
			"Do not claim GraphRAG, graph databases, or native vector search as required parts of the current production path",
			"Do not collapse research experiments into the mainline architecture without isolated evaluation",
		},
		ResearchTracks: []string{
			"Expand the controlled indexing benchmark to compare the relational baseline against vector retrieval",
			"Evaluate GraphRAG as an isolated experimental extension instead of a core dependency",
			"Strengthen grounded chat with measurable citation quality, accuracy, and latency metrics",
		},
		ArtifactRefs: []RuntimeProfileArtifact{
			{
				Label:   "Product focus RFC",
				Path:    "rfc/0001-product-focus.md",
				Kind:    "RFC",
				Summary: "Defines the primary use case, target users, explicit non-goals, and backlog implications.",
			},
			{
				Label:   "SBC template reference",
				Path:    "docs/article/sbc_template (1).doc",
				Kind:    "ARTICLE_TEMPLATE",
				Summary: "Original SBC Word template kept as the formatting reference for the final submission file.",
			},
			{
				Label:   "Academic workspace",
				Path:    "docs/article/README.md",
				Kind:    "ARTICLE_WORKSPACE",
				Summary: "Workspace notes and figure sources used to prepare the final SBC submission artifact outside the main runtime history.",
			},
			{
				Label:   "Indexing benchmark corpus",
				Path:    "api/internal/usecase/testdata/indexing-benchmark/corpus.json",
				Kind:    "BENCHMARK_CORPUS",
				Summary: "Controlled corpus and exact-match expectations used by the public indexing benchmark endpoint.",
			},
			{
				Label:   "Academic continuation proposal",
				Path:    "docs/proposta-continuacao-tcc.md",
				Kind:    "RESEARCH_GUIDE",
				Summary: "Research continuation notes that frame vector and GraphRAG work as explicit future experiments.",
			},
		},
		LastReviewedOn: "2026-03-22",
	}
}
