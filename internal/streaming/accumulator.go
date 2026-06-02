package streaming

import (
	"sort"

	"github.com/bloodf/g0router/internal/providers"
)

type Accumulator struct {
	id                string
	created           int64
	model             string
	usage             *providers.Usage
	systemFingerprint *string
	choices           map[int]*choiceState
}

type choiceState struct {
	index        int
	role         string
	content      string
	toolCalls    []providers.ToolCall
	finishReason *string
}

func NewAccumulator() *Accumulator {
	return &Accumulator{
		choices: make(map[int]*choiceState),
	}
}

func (a *Accumulator) AddChunk(chunk providers.StreamChunk) {
	if a.id == "" {
		a.id = chunk.ID
	}
	if a.created == 0 {
		a.created = chunk.Created
	}
	if a.model == "" {
		a.model = chunk.Model
	}
	if a.systemFingerprint == nil {
		a.systemFingerprint = chunk.SystemFingerprint
	}
	if chunk.Usage != nil {
		a.usage = chunk.Usage
	}

	for _, streamChoice := range chunk.Choices {
		choice := a.choice(streamChoice.Index)
		choice.append(streamChoice)
	}
}

func (a *Accumulator) Response() providers.ChatResponse {
	choices := make([]providers.Choice, 0, len(a.choices))
	indexes := make([]int, 0, len(a.choices))
	for index := range a.choices {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	for _, index := range indexes {
		state := a.choices[index]
		choices = append(choices, providers.Choice{
			Index: state.index,
			Message: providers.Message{
				Role:      state.role,
				Content:   state.content,
				ToolCalls: state.toolCalls,
			},
			FinishReason: state.finishReason,
		})
	}

	return providers.ChatResponse{
		ID:                a.id,
		Object:            "chat.completion",
		Created:           a.created,
		Model:             a.model,
		Choices:           choices,
		Usage:             a.usage,
		SystemFingerprint: a.systemFingerprint,
	}
}

func (a *Accumulator) Usage() *providers.Usage {
	return a.usage
}

func (a *Accumulator) choice(index int) *choiceState {
	choice, ok := a.choices[index]
	if ok {
		return choice
	}

	choice = &choiceState{index: index}
	a.choices[index] = choice
	return choice
}

func (c *choiceState) append(streamChoice providers.StreamChoice) {
	if streamChoice.Delta.Role != nil {
		c.role = *streamChoice.Delta.Role
	}
	if streamChoice.Delta.Content != nil {
		c.content += *streamChoice.Delta.Content
	}
	if streamChoice.FinishReason != nil {
		c.finishReason = streamChoice.FinishReason
	}
	for index, toolCall := range streamChoice.Delta.ToolCalls {
		c.appendToolCall(index, toolCall)
	}
}

func (c *choiceState) appendToolCall(index int, delta providers.ToolCall) {
	for len(c.toolCalls) <= index {
		c.toolCalls = append(c.toolCalls, providers.ToolCall{})
	}

	call := &c.toolCalls[index]
	if delta.ID != "" {
		call.ID = delta.ID
	}
	if delta.Type != "" {
		call.Type = delta.Type
	}
	if delta.Function.Name != "" {
		call.Function.Name = delta.Function.Name
	}
	call.Function.Arguments += delta.Function.Arguments
}
