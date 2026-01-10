#include <napi.h>
#include <string>
#include <cstring>

// Include the CGO header
extern "C" {
#include "libm9m.h"
}

// Helper to throw JavaScript errors
void ThrowM9MError(Napi::Env env, m9m_error_t* err) {
    std::string message = err->message ? err->message : "Unknown error";
    m9m_error_free(err);
    throw Napi::Error::New(env, message);
}

// Engine wrapper class
class Engine : public Napi::ObjectWrap<Engine> {
public:
    static Napi::Object Init(Napi::Env env, Napi::Object exports);
    Engine(const Napi::CallbackInfo& info);
    ~Engine();

private:
    static Napi::FunctionReference constructor;
    m9m_engine_t handle_;

    Napi::Value Execute(const Napi::CallbackInfo& info);
    Napi::Value RegisterNode(const Napi::CallbackInfo& info);

    friend class Workflow;
};

Napi::FunctionReference Engine::constructor;

// Workflow wrapper class
class Workflow : public Napi::ObjectWrap<Workflow> {
public:
    static Napi::Object Init(Napi::Env env, Napi::Object exports);
    static Napi::Object NewInstance(Napi::Env env, m9m_workflow_t handle);
    Workflow(const Napi::CallbackInfo& info);
    ~Workflow();

private:
    static Napi::FunctionReference constructor;
    m9m_workflow_t handle_;

    static Napi::Value FromJSON(const Napi::CallbackInfo& info);
    static Napi::Value FromFile(const Napi::CallbackInfo& info);
    Napi::Value ToJSON(const Napi::CallbackInfo& info);
    Napi::Value GetName(const Napi::CallbackInfo& info);
    Napi::Value GetId(const Napi::CallbackInfo& info);

    friend class Engine;
};

Napi::FunctionReference Workflow::constructor;

// CredentialManager wrapper class
class CredentialManager : public Napi::ObjectWrap<CredentialManager> {
public:
    static Napi::Object Init(Napi::Env env, Napi::Object exports);
    CredentialManager(const Napi::CallbackInfo& info);
    ~CredentialManager();

private:
    static Napi::FunctionReference constructor;
    m9m_credential_manager_t handle_;

    Napi::Value Store(const Napi::CallbackInfo& info);

    friend class Engine;
};

Napi::FunctionReference CredentialManager::constructor;

// Engine implementation
Napi::Object Engine::Init(Napi::Env env, Napi::Object exports) {
    Napi::Function func = DefineClass(env, "Engine", {
        InstanceMethod("execute", &Engine::Execute),
        InstanceMethod("registerNode", &Engine::RegisterNode),
    });

    constructor = Napi::Persistent(func);
    constructor.SuppressDestruct();

    exports.Set("Engine", func);
    return exports;
}

Engine::Engine(const Napi::CallbackInfo& info) : Napi::ObjectWrap<Engine>(info) {
    Napi::Env env = info.Env();
    m9m_error_t err = {0, nullptr};

    if (info.Length() > 0 && info[0].IsObject()) {
        // Check if a credential manager was passed
        Napi::Object obj = info[0].As<Napi::Object>();
        if (obj.InstanceOf(CredentialManager::constructor.Value())) {
            CredentialManager* cm = Napi::ObjectWrap<CredentialManager>::Unwrap(obj);
            handle_ = m9m_engine_new_with_credentials(cm->handle_, &err);
        } else {
            handle_ = m9m_engine_new(&err);
        }
    } else {
        handle_ = m9m_engine_new(&err);
    }

    if (err.code != 0) {
        ThrowM9MError(env, &err);
    }
}

Engine::~Engine() {
    if (handle_ != 0) {
        m9m_engine_free(handle_);
        handle_ = 0;
    }
}

Napi::Value Engine::Execute(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    if (info.Length() < 1 || !info[0].IsObject()) {
        throw Napi::TypeError::New(env, "Workflow object required");
    }

    Napi::Object workflowObj = info[0].As<Napi::Object>();
    if (!workflowObj.InstanceOf(Workflow::constructor.Value())) {
        throw Napi::TypeError::New(env, "First argument must be a Workflow");
    }

    Workflow* workflow = Napi::ObjectWrap<Workflow>::Unwrap(workflowObj);

    // Get optional input data
    std::string inputJson = "[]";
    if (info.Length() > 1 && info[1].IsArray()) {
        Napi::Array inputArray = info[1].As<Napi::Array>();
        Napi::Object json = env.Global().Get("JSON").As<Napi::Object>();
        Napi::Function stringify = json.Get("stringify").As<Napi::Function>();
        inputJson = stringify.Call(json, {inputArray}).As<Napi::String>().Utf8Value();
    }

    m9m_error_t err = {0, nullptr};
    m9m_result_t result = m9m_engine_execute(
        handle_,
        workflow->handle_,
        const_cast<char*>(inputJson.c_str()),
        static_cast<int>(inputJson.length()),
        &err
    );

    if (err.code != 0) {
        ThrowM9MError(env, &err);
    }

    // Get result as JSON
    char* resultJson = m9m_result_to_json(result, &err);
    m9m_result_free(result);

    if (err.code != 0) {
        ThrowM9MError(env, &err);
    }

    std::string resultStr = resultJson ? resultJson : "{}";
    if (resultJson) {
        m9m_free_string(resultJson);
    }

    // Parse JSON to JavaScript object
    Napi::Object json = env.Global().Get("JSON").As<Napi::Object>();
    Napi::Function parse = json.Get("parse").As<Napi::Function>();
    return parse.Call(json, {Napi::String::New(env, resultStr)});
}

Napi::Value Engine::RegisterNode(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    if (info.Length() < 2) {
        throw Napi::TypeError::New(env, "Node type and callback required");
    }

    if (!info[0].IsString()) {
        throw Napi::TypeError::New(env, "First argument must be a string (node type)");
    }

    if (!info[1].IsFunction()) {
        throw Napi::TypeError::New(env, "Second argument must be a function");
    }

    // Note: Full callback support requires async work queue handling
    // For now, we just acknowledge the registration
    // A full implementation would store the callback and set up a trampoline

    return env.Undefined();
}

// Workflow implementation
Napi::Object Workflow::Init(Napi::Env env, Napi::Object exports) {
    Napi::Function func = DefineClass(env, "Workflow", {
        StaticMethod("fromJSON", &Workflow::FromJSON),
        StaticMethod("fromFile", &Workflow::FromFile),
        InstanceMethod("toJSON", &Workflow::ToJSON),
        InstanceAccessor("name", &Workflow::GetName, nullptr),
        InstanceAccessor("id", &Workflow::GetId, nullptr),
    });

    constructor = Napi::Persistent(func);
    constructor.SuppressDestruct();

    exports.Set("Workflow", func);
    return exports;
}

Napi::Object Workflow::NewInstance(Napi::Env env, m9m_workflow_t handle) {
    Napi::Object obj = constructor.New({});
    Workflow* workflow = Napi::ObjectWrap<Workflow>::Unwrap(obj);
    workflow->handle_ = handle;
    return obj;
}

Workflow::Workflow(const Napi::CallbackInfo& info) : Napi::ObjectWrap<Workflow>(info) {
    handle_ = 0;
}

Workflow::~Workflow() {
    if (handle_ != 0) {
        m9m_workflow_free(handle_);
        handle_ = 0;
    }
}

Napi::Value Workflow::FromJSON(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    if (info.Length() < 1) {
        throw Napi::TypeError::New(env, "JSON string or object required");
    }

    std::string jsonStr;
    if (info[0].IsString()) {
        jsonStr = info[0].As<Napi::String>().Utf8Value();
    } else if (info[0].IsObject()) {
        Napi::Object json = env.Global().Get("JSON").As<Napi::Object>();
        Napi::Function stringify = json.Get("stringify").As<Napi::Function>();
        jsonStr = stringify.Call(json, {info[0]}).As<Napi::String>().Utf8Value();
    } else {
        throw Napi::TypeError::New(env, "Argument must be a string or object");
    }

    m9m_error_t err = {0, nullptr};
    m9m_workflow_t handle = m9m_workflow_from_json(
        const_cast<char*>(jsonStr.c_str()),
        static_cast<int>(jsonStr.length()),
        &err
    );

    if (err.code != 0) {
        ThrowM9MError(env, &err);
    }

    return NewInstance(env, handle);
}

Napi::Value Workflow::FromFile(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    if (info.Length() < 1 || !info[0].IsString()) {
        throw Napi::TypeError::New(env, "File path string required");
    }

    std::string path = info[0].As<Napi::String>().Utf8Value();

    m9m_error_t err = {0, nullptr};
    m9m_workflow_t handle = m9m_workflow_from_file(const_cast<char*>(path.c_str()), &err);

    if (err.code != 0) {
        ThrowM9MError(env, &err);
    }

    return NewInstance(env, handle);
}

Napi::Value Workflow::ToJSON(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    m9m_error_t err = {0, nullptr};
    char* jsonStr = m9m_workflow_to_json(handle_, &err);

    if (err.code != 0) {
        ThrowM9MError(env, &err);
    }

    std::string result = jsonStr ? jsonStr : "{}";
    if (jsonStr) {
        m9m_free_string(jsonStr);
    }

    // Parse JSON to JavaScript object
    Napi::Object json = env.Global().Get("JSON").As<Napi::Object>();
    Napi::Function parse = json.Get("parse").As<Napi::Function>();
    return parse.Call(json, {Napi::String::New(env, result)});
}

Napi::Value Workflow::GetName(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    m9m_error_t err = {0, nullptr};
    char* jsonStr = m9m_workflow_to_json(handle_, &err);

    if (err.code != 0) {
        return env.Null();
    }

    std::string result = jsonStr ? jsonStr : "{}";
    if (jsonStr) {
        m9m_free_string(jsonStr);
    }

    // Parse and extract name
    Napi::Object json = env.Global().Get("JSON").As<Napi::Object>();
    Napi::Function parse = json.Get("parse").As<Napi::Function>();
    Napi::Object parsed = parse.Call(json, {Napi::String::New(env, result)}).As<Napi::Object>();

    if (parsed.Has("name")) {
        return parsed.Get("name");
    }
    return env.Null();
}

Napi::Value Workflow::GetId(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    m9m_error_t err = {0, nullptr};
    char* jsonStr = m9m_workflow_to_json(handle_, &err);

    if (err.code != 0) {
        return env.Null();
    }

    std::string result = jsonStr ? jsonStr : "{}";
    if (jsonStr) {
        m9m_free_string(jsonStr);
    }

    // Parse and extract id
    Napi::Object json = env.Global().Get("JSON").As<Napi::Object>();
    Napi::Function parse = json.Get("parse").As<Napi::Function>();
    Napi::Object parsed = parse.Call(json, {Napi::String::New(env, result)}).As<Napi::Object>();

    if (parsed.Has("id")) {
        return parsed.Get("id");
    }
    return env.Null();
}

// CredentialManager implementation
Napi::Object CredentialManager::Init(Napi::Env env, Napi::Object exports) {
    Napi::Function func = DefineClass(env, "CredentialManager", {
        InstanceMethod("store", &CredentialManager::Store),
    });

    constructor = Napi::Persistent(func);
    constructor.SuppressDestruct();

    exports.Set("CredentialManager", func);
    return exports;
}

CredentialManager::CredentialManager(const Napi::CallbackInfo& info) : Napi::ObjectWrap<CredentialManager>(info) {
    Napi::Env env = info.Env();
    m9m_error_t err = {0, nullptr};

    handle_ = m9m_credential_manager_new(&err);

    if (err.code != 0) {
        ThrowM9MError(env, &err);
    }
}

CredentialManager::~CredentialManager() {
    if (handle_ != 0) {
        m9m_credential_manager_free(handle_);
        handle_ = 0;
    }
}

Napi::Value CredentialManager::Store(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();

    if (info.Length() < 1 || !info[0].IsObject()) {
        throw Napi::TypeError::New(env, "Credential object required");
    }

    Napi::Object json = env.Global().Get("JSON").As<Napi::Object>();
    Napi::Function stringify = json.Get("stringify").As<Napi::Function>();
    std::string credJson = stringify.Call(json, {info[0]}).As<Napi::String>().Utf8Value();

    m9m_error_t err = {0, nullptr};
    int result = m9m_credential_manager_store(
        handle_,
        const_cast<char*>(credJson.c_str()),
        static_cast<int>(credJson.length()),
        &err
    );

    if (err.code != 0) {
        ThrowM9MError(env, &err);
    }

    return Napi::Boolean::New(env, result != 0);
}

// Version function
Napi::Value GetVersion(const Napi::CallbackInfo& info) {
    Napi::Env env = info.Env();
    char* version = m9m_version();
    std::string result = version ? version : "unknown";
    if (version) {
        m9m_free_string(version);
    }
    return Napi::String::New(env, result);
}

// Module initialization
Napi::Object Init(Napi::Env env, Napi::Object exports) {
    Engine::Init(env, exports);
    Workflow::Init(env, exports);
    CredentialManager::Init(env, exports);
    exports.Set("version", Napi::Function::New(env, GetVersion));
    return exports;
}

NODE_API_MODULE(m9m, Init)
