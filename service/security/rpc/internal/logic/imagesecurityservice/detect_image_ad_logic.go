package imagesecurityservicelogic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"sea-try-go/service/common/logger"
	"sea-try-go/service/security/rpc/internal/svc"
	"sea-try-go/service/security/rpc/pb/sea-try-go/service/security/rpc/pb"
)

type DetectImageAdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDetectImageAdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DetectImageAdLogic {
	return &DetectImageAdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DashScopeResponse 阿里云 DashScope 响应格式
type DashScopeResponse struct {
	Output struct {
		Choices []struct {
			Message struct {
				Content interface{} `json:"content"` // 可能是 string 或 array
			} `json:"message"`
		} `json:"choices"`
	} `json:"output"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}

// AdDetectionResult 广告检测结果
type AdDetectionResult struct {
	IsAd          bool    `json:"is_ad"`
	AdConfidence  float64 `json:"ad_confidence"`
	ExtractedText string  `json:"extracted_text"`
	Success       bool    `json:"success"`
	ErrorMessage  string  `json:"error_message"`
}

// callAIModel 调用阿里云 DashScope 多模态 API 进行广告检测
func (l *DetectImageAdLogic) callAIModel(imageInput string, confidenceThreshold float64, enableTextExtraction bool) (*AdDetectionResult, error) {
	config := l.svcCtx.Config.AIModel

	// 构建提示词
	prompt := `请分析这张图片：
1. 判断是否为广告图片（包含商品推广、营销信息、联系方式等）
2. 提取图片中的文字内容（如果有）
3. 返回 JSON 格式：{"is_ad": true/false, "ad_confidence": 0.0-1.0, "extracted_text": "提取的文字"}

请严格只返回 JSON，不要其他内容。`

	// 构建 DashScope 请求（支持 URL 或 Base64 Data URI）
	dashScopeReq := map[string]interface{}{
		"model": "qwen-vl-max",
		"input": map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]interface{}{
						{"image": imageInput},
						{"text": prompt},
					},
				},
			},
		},
		"parameters": map[string]interface{}{
			"temperature": 0.1,
			"top_p":       0.8,
		},
	}

	requestBody, err := json.Marshal(dashScopeReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dashscope request: %w", err)
	}

	logger.LogInfo(l.ctx, "Calling DashScope API", logger.WithUserID(fmt.Sprintf("endpoint: %s", config.ModelEndpoint)))

	// 创建独立的超时上下文，避免被上层 gRPC 上下文过早取消
	apiCtx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	httpReq, err := http.NewRequestWithContext(apiCtx, "POST", config.ModelEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.APIKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call DashScope API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	logger.LogInfo(l.ctx, "DashScope API response", logger.WithUserID(fmt.Sprintf("status: %d, body: %s", resp.StatusCode, string(body))))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DashScope API returned status %d: %s", resp.StatusCode, string(body))
	}

	var dashScopeResp DashScopeResponse
	if err := json.Unmarshal(body, &dashScopeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DashScope response: %w", err)
	}

	if len(dashScopeResp.Output.Choices) == 0 {
		return nil, fmt.Errorf("no choices in DashScope response")
	}

	content := dashScopeResp.Output.Choices[0].Message.Content
	logger.LogInfo(l.ctx, "AI response content", logger.WithUserID(fmt.Sprintf("content: %v", content)))

	// 提取 JSON（content 可能是 string 或 array）
	var contentStr string
	switch v := content.(type) {
	case string:
		contentStr = v
	case []interface{}:
		// 多模态响应格式：content 是数组
		if len(v) > 0 {
			if text, ok := v[0].(map[string]interface{}); ok {
				contentStr = fmt.Sprintf("%v", text["text"])
			}
		}
	default:
		contentStr = fmt.Sprintf("%v", v)
	}

	// 提取 JSON（可能包含在 markdown 代码块中）
	jsonStr := extractJSON(contentStr)

	var result AdDetectionResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		logger.LogInfo(l.ctx, "JSON parse failed, using fallback", logger.WithUserID(fmt.Sprintf("error: %v, content: %s", err, contentStr)))
		result = parseAdResultFallback(contentStr)
	}

	result.Success = true
	return &result, nil
}

// extractJSON 从文本中提取 JSON 字符串
func extractJSON(text string) string {
	start := strings.Index(text, "```")
	if start != -1 {
		end := strings.Index(text[start+3:], "```")
		if end != -1 {
			content := text[start+3 : start+3+end]
			content = strings.TrimPrefix(content, "json")
			return strings.TrimSpace(content)
		}
	}
	return strings.TrimSpace(text)
}

// parseAdResultFallback 回退解析函数
func parseAdResultFallback(content string) AdDetectionResult {
	result := AdDetectionResult{
		IsAd:          strings.Contains(strings.ToLower(content), "是广告") || strings.Contains(strings.ToLower(content), `"is_ad": true`),
		AdConfidence:  0.5,
		ExtractedText: content,
	}
	if strings.Contains(strings.ToLower(content), "高") || strings.Contains(strings.ToLower(content), "high") {
		result.AdConfidence = 0.9
	} else if strings.Contains(strings.ToLower(content), "低") || strings.Contains(strings.ToLower(content), "low") {
		result.AdConfidence = 0.3
	}
	return result
}

// DetectImageAd 广告检测接口
func (l *DetectImageAdLogic) DetectImageAd(in *pb.DetectImageAdRequest) (*pb.DetectImageAdResponse, error) {
	// 验证输入参数
	if in.ImageUrl == "" && in.ImageBase64 == "" {
		return &pb.DetectImageAdResponse{
			Success:      false,
			ErrorMessage: "image_url or image_base64 is required",
		}, nil
	}

	// 设置默认选项
	confidenceThreshold := float64(0.7)
	enableTextExtraction := true

	if in.Options != nil {
		if in.Options.ConfidenceThreshold > 0 {
			confidenceThreshold = float64(in.Options.ConfidenceThreshold)
		}
		enableTextExtraction = in.Options.EnableTextExtraction
	}

	// 优先使用 Base64 数据，否则使用 URL
	imageInput := in.ImageUrl
	if in.ImageBase64 != "" {
		imageInput = in.ImageBase64
	}

	// 调用 AI 模型服务
	modelResp, err := l.callAIModel(imageInput, confidenceThreshold, enableTextExtraction)
	if err != nil {
		logger.LogBusinessErr(l.ctx, 500, err)
		return &pb.DetectImageAdResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("AI model service error: %v", err),
		}, nil
	}

	// 构建响应
	response := &pb.DetectImageAdResponse{
		IsAd:          modelResp.IsAd,
		AdConfidence:  float32(modelResp.AdConfidence),
		ExtractedText: modelResp.ExtractedText,
		Success:       true,
	}

	logger.LogInfo(l.ctx, "Image ad detection completed",
		logger.WithUserID(fmt.Sprintf("IsAd:%v, Confidence:%.2f", response.IsAd, response.AdConfidence)))

	return response, nil
}