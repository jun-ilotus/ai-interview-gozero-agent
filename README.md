## 快速启动
### 后端
1. 下载Docker
2. 执行命令
### 启动项目
```
docker-compose up -d
```
### 停止项目
```
docker-compose down -v
```
> 无需配置任何环境，Docker一键部署，包括建表

## 功能内容：
1. 交互流程与状态管理
   - 接收客户端 Post 请求（含用户输入 / 操作指令 / 文件上传） 
   - 通过 SSE 流式实时输出 AI 面试问题 / 反馈 
   - 利用 Redis 状态机（SessionID:State）管理面试流程，AI 能主动引导话题，推进面试目标维持请求 - 响应链路的低延迟交互
2. 多轮对话与知识库管理
   - 基于 pgvector 扩展的 vector_store 表存储对话数据及知识库内容（含 id/chat_id(or doc_id)/role/content/embedding/created_at 字段） 
   - 支持单条消息/知识存储与历史对话/知识批量查询（通过 chat_id/doc_id 关联） 
   - 依托 embedding 向量实现对话上下文关联、连续性维护及知识库检索
3. PDF 处理与知识库构建
   - 通过 MCP 服务（gRPC）接收并解析客户端上传的 PDF 文件，转换为文字内容并生成向量 
   - 提供独立 POST 接口，支持上传 PDF 文件至 RAG 本地知识库（存储原始文本及向量至 pgvector） 
   - 在 SSE 聊天交互中，自动将知识库检索结果与当前解析文本（如有）拼接至上下文，作为 AI 生成响应的参考依据
4. RAG 本地知识库集成
   - 支持构建本地知识库（通过专用接口上传 PDF 向量化存储） 
   - 基于用户输入、对话历史及知识库内容，实时检索（向量相似度）知识库中相关内容辅助生成回复，提升 AI 响应的专业性与针对性
5. 智能体调度与部署
   - 基于 Redis 状态机实现 AI 智能体的目标导向行为，动态调整面试流程 
   - 采用容器化部署：通过 Dockerfile 构建镜像，docker-compose.yml 编排服务（API, MCP-gRPC, PostgreSQL-pgvector, Redis, etcd），init.sql 初始化数据库表结构及扩展 
   - 实现一键启动：本地安装 Docker 后，执行 docker-compose up 即可启动全套服务（API、MCP、DB、Redis、etcd），无需额外环境配置