package scheduler

import (
	"testing"
)

func TestConfig_BuildStrategyChain(t *testing.T) {
	tests := []struct {
		name      string
		chain     []string
		wantNames []string
	}{
		{
			name:      "默认策略链",
			chain:     []string{"direct", "affinity", "label_match"},
			wantNames: []string{"direct", "affinity", "label_match"},
		},
		{
			name:      "自定义策略链",
			chain:     []string{"load_balance", "random"},
			wantNames: []string{"load_balance", "random"},
		},
		{
			name:      "空策略链使用默认",
			chain:     []string{},
			wantNames: []string{"label_match"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Strategy.Chain = tt.chain

			chain := cfg.BuildStrategyChain()
			strategies := chain.Strategies()

			if len(strategies) != len(tt.wantNames) {
				t.Errorf("expected %d strategies, got %d", len(tt.wantNames), len(strategies))
				return
			}

			for i, want := range tt.wantNames {
				if strategies[i].Name() != want {
					t.Errorf("strategy %d: expected %s, got %s", i, want, strategies[i].Name())
				}
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := &Config{}
	cfg.Validate()

	if cfg.NodeID == "" {
		t.Error("expected NodeID to be set")
	}
	if cfg.Strategy.Default == "" {
		t.Error("expected Strategy.Default to be set")
	}
	if len(cfg.Strategy.Chain) == 0 {
		t.Error("expected Strategy.Chain to be set")
	}
	if cfg.Redis.ReadTimeout == 0 {
		t.Error("expected Redis.ReadTimeout to be set")
	}
}
