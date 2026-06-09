package schemas

// FileObject represents a file in the OpenAI files API.
type FileObject struct {
	ID            string  `json:"id"`
	Object        string  `json:"object"`
	Bytes         int     `json:"bytes"`
	CreatedAt     int64   `json:"created_at"`
	Filename      string  `json:"filename"`
	Purpose       string  `json:"purpose"`
	Status        string  `json:"status,omitempty"`
	StatusDetails *string `json:"status_details,omitempty"`
}

// FileUploadRequest is the payload for POST /v1/files.
type FileUploadRequest struct {
	File     []byte `json:"-"`
	Filename string `json:"-"`
	Purpose  string `json:"purpose"`
}

// FileListResponse is the response for GET /v1/files.
type FileListResponse struct {
	Object string     `json:"object"`
	Data   []FileObject `json:"data"`
}

// FileDeleteResponse is the response for DELETE /v1/files/{id}.
type FileDeleteResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}
