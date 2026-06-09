package schemas

// Batch represents a batch job in the OpenAI batch API.
type Batch struct {
	ID               string             `json:"id"`
	Object           string             `json:"object"`
	Endpoint         string             `json:"endpoint"`
	Errors           *BatchErrors       `json:"errors,omitempty"`
	InputFileID      string             `json:"input_file_id"`
	CompletionWindow string             `json:"completion_window"`
	Status           string             `json:"status"`
	OutputFileID     *string            `json:"output_file_id,omitempty"`
	ErrorFileID      *string            `json:"error_file_id,omitempty"`
	CreatedAt        int64              `json:"created_at"`
	InProgressAt     *int64             `json:"in_progress_at,omitempty"`
	CompletedAt      *int64             `json:"completed_at,omitempty"`
	ExpiredAt        *int64             `json:"expired_at,omitempty"`
	CancellingAt     *int64             `json:"cancelling_at,omitempty"`
	CancelledAt      *int64             `json:"cancelled_at,omitempty"`
	RequestCounts    *BatchRequestCounts `json:"request_counts,omitempty"`
	Metadata         map[string]string  `json:"metadata,omitempty"`
}

// BatchErrors holds a list of batch-level errors.
type BatchErrors struct {
	Object string       `json:"object"`
	Data   []BatchError `json:"data"`
}

// BatchError is a single error inside a batch.
type BatchError struct {
	Line    *int    `json:"line,omitempty"`
	Message string  `json:"message"`
	Param   *string `json:"param,omitempty"`
	Code    *string `json:"code,omitempty"`
}

// BatchRequestCounts tracks the number of requests in a batch by outcome.
type BatchRequestCounts struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// BatchCreateRequest is the payload for POST /v1/batches.
type BatchCreateRequest struct {
	InputFileID      string            `json:"input_file_id"`
	Endpoint         string            `json:"endpoint"`
	CompletionWindow string            `json:"completion_window"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

// BatchListResponse is the response for GET /v1/batches.
type BatchListResponse struct {
	Object string  `json:"object"`
	Data   []Batch `json:"data"`
}
