package logic

import (
	"ai-gozero-agent/api/internal/utils"
	"context"
	"fmt"

	"ai-gozero-agent/api/internal/svc"
	"ai-gozero-agent/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type KnowledgeUploadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 知识库上传
func NewKnowledgeUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *KnowledgeUploadLogic {
	return &KnowledgeUploadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *KnowledgeUploadLogic) KnowledgeUpload(req *types.KnowledgeUploadReq) (resp *types.KnowledgeUploadResp, err error) {
	fmt.Println("进入logic处理！！：")
	// 分块处理知识库内容
	chunks := utils.SplitText(req.Content, l.svcCtx.Config.VectorDB.Knowledge.MaxChunkSize)
	fmt.Println("准备分块！！:")
	// 保存每个分块
	for _, chunk := range chunks {
		if err := l.svcCtx.VectorStore.SaveKnowledge(req.Title, chunk, l.svcCtx.Config.VectorDB); err != nil {
			logx.Errorf("save knowledge failed: %v", err)
			return nil, err
		}
	}
	fmt.Println("保存保存结束！！:")

	return &types.KnowledgeUploadResp{
		Msg:    "知识上传成功",
		Chunks: len(chunks),
	}, nil
}
