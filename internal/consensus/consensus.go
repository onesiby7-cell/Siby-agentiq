package consensus

import (
	"context"
	"sync"
	"time"
)

const (
	ConsensusTimeout = 3 * time.Second
	QuorumRequired   = 2
)

type ConsensusEngine struct {
	agents  []ValidatorAgent
	mu      sync.RWMutex
	timeout time.Duration
	quorum  int
}

type ValidatorAgent struct {
	Name     string
	Type     AgentType
	Validate func(response string) ValidationResult
}

type AgentType string

const (
	ArchitectAgent AgentType = "ARCHITECT"
	ThinkerAgent   AgentType = "THINKER"
	StylistAgent   AgentType = "STYLIST"
)

type ValidationResult struct {
	Valid        bool
	Issues       []string
	Severity     Severity
	SuggestedFix string
}

type Severity string

const (
	SeverityError   Severity = "ERROR"
	SeverityWarning Severity = "WARNING"
	SeverityInfo    Severity = "INFO"
)

type ConsensusResult struct {
	Approved    bool
	Validations []ValidationResult
	QuorumMet   bool
	SelfHealing bool
	FixedResult string
	Duration    time.Duration
}

func NewConsensusEngine() *ConsensusEngine {
	engine := &ConsensusEngine{
		agents:  make([]ValidatorAgent, 0),
		timeout: ConsensusTimeout,
		quorum:  QuorumRequired,
	}
	engine.initAgents()
	return engine
}

func (ce *ConsensusEngine) initAgents() {
	ce.agents = []ValidatorAgent{
		{
			Name: "Architect",
			Type: ArchitectAgent,
			Validate: func(response string) ValidationResult {
				return validateArchitecture(response)
			},
		},
		{
			Name: "Thinker",
			Type: ThinkerAgent,
			Validate: func(response string) ValidationResult {
				return validateLogic(response)
			},
		},
		{
			Name: "Stylist",
			Type: StylistAgent,
			Validate: func(response string) ValidationResult {
				return validateStyle(response)
			},
		},
	}
}

func (ce *ConsensusEngine) Validate(ctx context.Context, response string) *ConsensusResult {
	result := &ConsensusResult{
		Validations: make([]ValidationResult, 0),
	}

	start := time.Now()

	validations := make(chan ValidationResult, len(ce.agents))
	var wg sync.WaitGroup

	for _, agent := range ce.agents {
		wg.Add(1)
		go func(a ValidatorAgent) {
			defer wg.Done()
			validation := a.Validate(response)
			validations <- validation
		}(agent)
	}

	go func() {
		wg.Wait()
		close(validations)
	}()

	for v := range validations {
		result.Validations = append(result.Validations, v)
	}

	validCount := 0
	for _, v := range result.Validations {
		if v.Valid {
			validCount++
		}
	}

	result.QuorumMet = validCount >= ce.quorum
	result.Approved = result.QuorumMet

	if !result.Approved {
		result.SelfHealing = true
		result.FixedResult = ce.selfHeal(response, result.Validations)
	}

	result.Duration = time.Since(start)

	return result
}

func (ce *ConsensusEngine) selfHeal(response string, validations []ValidationResult) string {
	var fixes []string
	for _, v := range validations {
		if !v.Valid && v.SuggestedFix != "" {
			fixes = append(fixes, v.SuggestedFix)
		}
	}

	if len(fixes) > 0 {
		return "Self-healed: " + fixes[0]
	}
	return response
}

func validateArchitecture(response string) ValidationResult {
	result := ValidationResult{Valid: true}

	if len(response) == 0 {
		result.Valid = false
		result.Issues = append(result.Issues, "Empty response")
		result.Severity = SeverityError
		return result
	}

	return result
}

func validateLogic(response string) ValidationResult {
	result := ValidationResult{Valid: true}

	errorPatterns := []string{"undefined", "cannot find", "syntax error", "null pointer"}
	for _, pattern := range errorPatterns {
		if contains(response, pattern) {
			result.Valid = false
			result.Issues = append(result.Issues, "Logic error detected: "+pattern)
			result.Severity = SeverityError
			result.SuggestedFix = "Auto-fix: Check variable declarations"
			break
		}
	}

	return result
}

func validateStyle(response string) ValidationResult {
	result := ValidationResult{Valid: true}

	if len(response) > 10000 {
		result.Valid = false
		result.Issues = append(result.Issues, "Response too long")
		result.Severity = SeverityWarning
		result.SuggestedFix = "Optimize: Simplify the response"
	}

	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (ce *ConsensusEngine) GetAgents() []ValidatorAgent {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	return ce.agents
}

func (ce *ConsensusEngine) SetQuorum(q int) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.quorum = q
}

func (ce *ConsensusEngine) SetTimeout(t time.Duration) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.timeout = t
}
