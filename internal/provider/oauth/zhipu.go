package oauth

import (
	"context"
	"errors"
)

const (
	zhipuProviderID ProviderID = "zhipu"
	zhipuAuthURL               = "https://open.bigmodel.cn/usercenter/apikeys"
)

// ZhipuFlow implements Zhipu direct API-key credential capture.
type ZhipuFlow struct {
	apiKey *apiKeyFlow
}

// NewZhipuFlow returns a Zhipu API-key compatible OAuth flow.
func NewZhipuFlow() *ZhipuFlow {
	return &ZhipuFlow{
		apiKey: newAPIKeyFlow(zhipuProviderID, zhipuAuthURL),
	}
}

func (f *ZhipuFlow) ProviderID() ProviderID {
	return zhipuProviderID
}

func (f *ZhipuFlow) Start(ctx context.Context) (AuthSession, error) {
	return f.apiKey.start(ctx)
}

func (f *ZhipuFlow) Exchange(ctx context.Context, session AuthSession, code string) (TokenResult, error) {
	return f.apiKey.exchange(ctx, session, code)
}

func (f *ZhipuFlow) Poll(ctx context.Context, session AuthSession) (PollResult, error) {
	return PollResult{}, errors.New("zhipu api key flow does not support poll")
}
