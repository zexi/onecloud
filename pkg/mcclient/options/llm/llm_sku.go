package llm

import (
	"strings"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type LLMSkuListOptions struct {
	options.BaseListOptions

	LLMType    string `json:"llm_type" choices:"ollama|vllm|comfyui|openclaw|sglang"`
	Source     string `json:"source" help:"filter by source (huggingface, model_scope, local_path)"`
	Categories string `json:"categories" help:"filter by category (llm, embedding, image, ...)"`
}

func (o *LLMSkuListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type LLMSkuShowOptions struct {
	options.BaseShowOptions
}

func (o *LLMSkuShowOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMSkuCreateOptions struct {
	LLMSkuBaseCreateOptions

	MountedModels []string `help:"mounted models, <model_id> e.g. qwen2:0.5b-dup" json:"mounted_models"`

	LLM_IMAGE_ID string `json:"llm_image_id"`
	LLM_TYPE     string `json:"llm_type" choices:"ollama|vllm|comfyui|sglang"`

	// Model source
	Source              string `help:"model source: huggingface, model_scope, local_path" json:"source"`
	HuggingfaceRepoId  string `help:"HuggingFace repo ID" json:"huggingface_repo_id"`
	HuggingfaceFilename string `help:"HuggingFace filename" json:"huggingface_filename"`
	ModelScopeModelId   string `help:"ModelScope model ID" json:"model_scope_model_id"`
	LocalPath           string `help:"local model path" json:"local_path"`
	// Model metadata
	Categories      string   `help:"model categories, comma-separated: llm,embedding,image" json:"-"`
	BackendVersion  string   `help:"inference backend version" json:"backend_version"`

	PreferredModel string   `help:"preferred model (vllm only), sets llm_spec.vllm.preferred_model" json:"-"`
	VllmArg        []string `help:"vLLM args in format key=value; use key= for flags without values" json:"-"`
}

func (o *LLMSkuCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)
	if err := o.LLMSkuBaseCreateOptions.Params(dict); err != nil {
		return nil, err
	}
	fetchMountedModels(o.MountedModels, dict)
	if len(o.Categories) > 0 {
		cats := jsonutils.NewArray()
		for _, c := range strings.Split(o.Categories, ",") {
			cats.Add(jsonutils.NewString(strings.TrimSpace(c)))
		}
		dict.Set("categories", cats)
	}
	vllmSpec, err := newVLLMSpecFromArgs(o.PreferredModel, o.VllmArg)
	if err != nil {
		return nil, err
	}
	if o.LLM_TYPE == string(api.LLM_CONTAINER_VLLM) && vllmSpec != nil {
		spec := &api.LLMSpec{
			Ollama: nil,
			Vllm:   vllmSpec,
			Dify:   nil,
		}
		dict.Set("llm_spec", jsonutils.Marshal(spec))
	}
	return dict, nil
}

type LLMSkuDeleteOptions struct {
	options.BaseIdOptions
}

func (o *LLMSkuDeleteOptions) GetId() string {
	return o.ID
}

func (o *LLMSkuDeleteOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}

type LLMSkuUpdateOptions struct {
	LLMSkuBaseUpdateOptions

	MountedModels []string `help:"mounted models, <model_id> e.g. qwen2:0.5b-dup" json:"mounted_models"`

	// For ollama/vllm; backend merges into LLMSpec. Use dify-sku update for dify type.
	LlmImageId string `json:"llm_image_id"`

	PreferredModel string   `help:"preferred model (vllm only), sets llm_spec.vllm.preferred_model" json:"-"`
	VllmArg        []string `help:"vLLM args in format key=value; use key= for flags without values" json:"-"`
}

func (o *LLMSkuUpdateOptions) GetId() string {
	return o.ID
}

func (o *LLMSkuUpdateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	obj := jsonutils.Marshal(o)
	obj.Unmarshal(dict)
	if err := o.LLMSkuBaseUpdateOptions.Params(dict); err != nil {
		return nil, err
	}
	fetchMountedModels(o.MountedModels, dict)
	vllmSpec, err := newVLLMSpecFromArgs(o.PreferredModel, o.VllmArg)
	if err != nil {
		return nil, err
	}
	if vllmSpec != nil {
		spec := &api.LLMSpec{
			Ollama: nil,
			Vllm:   vllmSpec,
			Dify:   nil,
		}
		dict.Set("llm_spec", jsonutils.Marshal(spec))
	}
	return dict, nil
}
