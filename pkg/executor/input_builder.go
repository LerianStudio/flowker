// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package executor

// InputBuilder is an optional interface that providers can implement
// to customize how ExecutionInput is built from provider configuration.
// If a provider implements InputBuilder, the runtime uses it instead
// of the generic buildProviderConfigRunnerInput.
type InputBuilder interface {
	BuildInput(providerConfig map[string]any, executorID ID, nodeData map[string]any, requestBody []byte) (ExecutionInput, error)
}
