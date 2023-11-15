package upload

// precreate 参数
type PrecreateArg struct {
	Path      string   `json:"path"`
	Size      uint64   `json:"size"`
	BlockList []string `json:"block_list"`
}

// 创建 PrecreateArg 实例
func NewPrecreateArg(path string, size uint64, blockList []string) *PrecreateArg {
	s := new(PrecreateArg)
	s.Path = path
	s.Size = size
	s.BlockList = blockList
	return s
}

// PrecreateReturn
type PrecreateReturn struct {
	Errno      int    `json:"errno"`
	ReturnType int    `json:"return_type"`
	BlockList  []int  `json:"block_list"` // 分片序号列表
	UploadId   string `json:"uploadid"`
	RequestId  int    `json:"request_id"`
}

// upload 参数
type UploadArg struct {
	UploadId  string `json:"uploadid"`
	Path      string `json:"path"`
	LocalFile string `json:"local_file"`
	Partseq   int    `json:"partseq"`
}

// 创建 UploadArg 实例
func NewUploadArg(uploadId string, path string, localFile string, partseq int) *UploadArg {
	s := new(UploadArg)
	s.UploadId = uploadId
	s.Path = path
	s.LocalFile = localFile
	s.Partseq = partseq
	return s
}

// UploadReturn
type UploadReturn struct {
	Md5       string `json:"md5"`
	RequestId int    `json:"request_id"`
}

// create 参数
type CreateArg struct {
	UploadId  string   `json:"uploadid"`
	Path      string   `json:"path"`
	Size      uint64   `json:"size"`
	BlockList []string `json:"block_list"`
}

// 创建 CreateArg 实例
func NewCreateArg(uploadId string, path string, size uint64, blockList []string) *CreateArg {
	s := new(CreateArg)
	s.UploadId = uploadId
	s.Path = path
	s.Size = size
	s.BlockList = blockList
	return s
}

// CreateReturn
type CreateReturn struct {
	Errno int    `json:"errno"`
	Path  string `json:"path"`
}
