package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func createFile(name string, content string) (remove func(), err error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return nil, err
	}

	return func() {
		if err := os.Remove(name); err != nil {
			panic(fmt.Sprintf("failed to remove file: %s", err))
		}
	}, nil
}

func setEnvVariables(vals map[string]string) func() {
	for k, v := range vals {
		os.Setenv(k, v)
	}
	return func() {
		for k := range vals {
			os.Unsetenv(k)
		}
	}
}

func TestLoadEnvConfig(t *testing.T) {
	for label, test := range map[string]struct {
		fileContent string
		envVars     map[string]string
		want        *EnvConfig
	}{
		"all values empty in file and env": {
			fileContent: `TELEGRAM_ID=
TELEGRAM_TOKEN=
EDIT_WAIT_SECONDS=
OPENAI_API_KEY=
PROMPT_INIT=`,
			want: &EnvConfig{
				TelegramID:      []int64{},
				TelegramToken:   "",
				EditWaitSeconds: 0,
				OpenAIKey:       "",
				PromptInit:      "",
			},
		},
		"no file, all values through env": {
			envVars: map[string]string{
				"TELEGRAM_ID":       "123,456",
				"TELEGRAM_TOKEN":    "token",
				"EDIT_WAIT_SECONDS": "10",
				"OPENAI_API_KEY":    "APIKEY",
				"PROMPT_INIT":       "dodo",
			},
			want: &EnvConfig{
				TelegramID:      []int64{123, 456},
				TelegramToken:   "token",
				EditWaitSeconds: 10,
				OpenAIKey:       "APIKEY",
				PromptInit:      "dodo",
			},
		},
		"all values provided in file, single TELEGRAM_ID": {
			fileContent: `TELEGRAM_ID=123
TELEGRAM_TOKEN=abc
EDIT_WAIT_SECONDS=10
OPENAI_API_KEY=xxx
PROMPT_INIT=dodo`,
			want: &EnvConfig{
				TelegramID:      []int64{123},
				TelegramToken:   "abc",
				EditWaitSeconds: 10,
				OpenAIKey:       "xxx",
				PromptInit:      "dodo",
			},
		},
		"multiple TELEGRAM_IDs provided in file": {
			fileContent: `TELEGRAM_ID=123,456
TELEGRAM_TOKEN=abc
EDIT_WAIT_SECONDS=10
OPENAI_API_KEY=xxx
PROMPT_INIT=dodo`,
			envVars: map[string]string{},
			want: &EnvConfig{
				TelegramID:      []int64{123, 456},
				TelegramToken:   "abc",
				EditWaitSeconds: 10,
				OpenAIKey:       "xxx",
				PromptInit:      "dodo",
			},
		},
		"env variables should override file values": {
			fileContent: `TELEGRAM_ID=123
TELEGRAM_TOKEN=abc
EDIT_WAIT_SECONDS=10
OPENAI_API_KEY=xxx
PROMPT_INIT=dodo`,
			envVars: map[string]string{
				"TELEGRAM_ID":       "456",
				"TELEGRAM_TOKEN":    "def",
				"EDIT_WAIT_SECONDS": "20",
				"OPENAI_API_KEY":    "OOO",
				"PROMPT_INIT":       "soso",
			},
			want: &EnvConfig{
				TelegramID:      []int64{456},
				TelegramToken:   "def",
				EditWaitSeconds: 20,
				OpenAIKey:       "OOO",
				PromptInit:      "soso",
			},
		},
		"multiple TELEGRAM_IDs provided in env": {
			fileContent: `TELEGRAM_ID=123
TELEGRAM_TOKEN=abc
EDIT_WAIT_SECONDS=10
OPENAI_API_KEY=xxx
PROMPT_INIT=dodo`,
			envVars: map[string]string{
				"TELEGRAM_ID": "456,789",
			},
			want: &EnvConfig{
				TelegramID:      []int64{456, 789},
				TelegramToken:   "abc",
				EditWaitSeconds: 10,
				OpenAIKey:       "xxx",
				PromptInit:      "dodo",
			},
		},
	} {
		t.Run(label, func(t *testing.T) {
			unset := setEnvVariables(test.envVars)
			t.Cleanup(unset)

			if test.fileContent != "" {
				remove, err := createFile("test.env", test.fileContent)
				require.NoError(t, err)
				t.Cleanup(remove)
			}

			cfg, err := LoadEnvConfig("test.env")
			require.NoError(t, err)
			require.Equal(t, test.want, cfg)
		})
	}
}
