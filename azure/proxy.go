package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/stulzq/azure-openai-proxy/util"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// Proxy Azure OpenAI
func Proxy(c *gin.Context) {
	director := func(req *http.Request) {
		if req.Body == nil {
			util.SendError(c, errors.New("request body is empty"))
			return
		}
		body, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(body))

		x := map[string]json.RawMessage{}
		err := json.Unmarshal(body, &x)
		if err != nil {
			util.SendError(c, errors.Wrap(err, "get model error"))
			return
		}

		// get deployment from request
		model := x["model"]
		// model to string
		var deployment string
		err = json.Unmarshal(model, &deployment)
		deployment = GetDeploymentByModel(deployment)

		// get auth token from header
		rawToken := req.Header.Get("Authorization")
		token := strings.TrimPrefix(rawToken, "Bearer ")
		req.Header.Set(AuthHeaderKey, token)
		req.Header.Del("Authorization")

		originURL := req.URL.String()
		req.Host = AzureOpenAIEndpointParse.Host
		req.URL.Scheme = AzureOpenAIEndpointParse.Scheme
		req.URL.Host = AzureOpenAIEndpointParse.Host
		req.URL.Path = path.Join(fmt.Sprintf("/openai/deployments/%s", deployment), strings.Replace(req.URL.Path, "/v1/", "/", 1))
		req.URL.RawPath = req.URL.EscapedPath()

		query := req.URL.Query()
		query.Add("api-version", AzureOpenAIAPIVer)
		req.URL.RawQuery = query.Encode()

		log.Printf("proxying request [%s] %s -> %s", model, originURL, req.URL.String())
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ServeHTTP(c.Writer, c.Request)

	// issue: https://github.com/Chanzhaoyu/chatgpt-web/issues/831
	if c.Writer.Header().Get("Content-Type") == "text/event-stream" {
		if _, err := c.Writer.Write([]byte{'\n'}); err != nil {
			log.Printf("rewrite response error: %v", err)
		}
	}
}

func GetDeploymentByModel(model string) string {
	if v, ok := AzureOpenAIModelMapper[model]; ok {
		return v
	}

	return fallbackModelMapper.ReplaceAllString(model, "")
}
