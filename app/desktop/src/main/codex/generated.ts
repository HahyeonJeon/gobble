// Generated from official Codex 0.153.4. Do not edit.
// Regenerate: npm run codex:schema. Empty environment lists are an app restriction.

export type AbsolutePathBuf = string;

export type AskForApproval =
  | 'untrusted'
  | 'on-request'
  | {
      granular: {
        sandbox_approval: boolean;
        rules: boolean;
        skill_approval: boolean;
        request_permissions: boolean;
        mcp_elicitations: boolean;
      };
    }
  | 'never';

export type ByteRange = {
  start: number;
  end: number;
};

export type DynamicToolCallOutputContentItem =
  | {
      type: 'inputText';
      text: string;
    }
  | {
      type: 'inputImage';
      imageUrl: string;
    }
  | {
      type: 'inputAudio';
      audioUrl: string;
    };

export type DynamicToolCallParams = {
  threadId: string;
  turnId: string;
  callId: string;
  namespace: string | null;
  tool: string;
  arguments: JsonValue;
};

export type DynamicToolCallResponse = {
  contentItems: Array<DynamicToolCallOutputContentItem>;
  success: boolean;
};

export type DynamicToolFunctionSpec = {
  name: string;
  description: string;
  inputSchema: JsonValue;
  deferLoading?: boolean;
};

export type DynamicToolNamespaceSpec = {
  name: string;
  description: string;
  tools: Array<DynamicToolNamespaceTool>;
};

export type DynamicToolNamespaceTool = {
  type: 'function';
} & DynamicToolFunctionSpec;

export type DynamicToolSpec =
  | ({
      type: 'function';
    } & DynamicToolFunctionSpec)
  | ({
      type: 'namespace';
    } & DynamicToolNamespaceSpec);

export type ImageDetail = 'auto' | 'low' | 'high' | 'original';

export type JsonValue =
  | number
  | string
  | boolean
  | Array<JsonValue>
  | {
      [key in string]?: JsonValue;
    }
  | null;

export type NetworkAccess = 'restricted' | 'enabled';

export type ReasoningEffort = string;

export type SandboxMode = 'read-only' | 'workspace-write' | 'danger-full-access';

export type SandboxPolicy =
  | {
      type: 'dangerFullAccess';
    }
  | {
      type: 'readOnly';
      networkAccess: boolean;
    }
  | {
      type: 'externalSandbox';
      networkAccess: NetworkAccess;
    }
  | {
      type: 'workspaceWrite';
      writableRoots: Array<AbsolutePathBuf>;
      networkAccess: boolean;
      excludeTmpdirEnvVar: boolean;
      excludeSlashTmp: boolean;
    };

export type TextElement = {
  byteRange: ByteRange;
  placeholder: string | null;
};

export type ThreadResumeParams = {
  threadId: string;
  model?: string | null;
  modelProvider?: string | null;
  cwd?: string | null;
  approvalPolicy?: AskForApproval | null;
  sandbox?: SandboxMode | null;
  config?:
    | {
        [key in string]?: JsonValue;
      }
    | null;
  developerInstructions?: string | null;
};

export type ThreadStartParams = {
  model?: string | null;
  modelProvider?: string | null;
  cwd?: string | null;
  approvalPolicy?: AskForApproval | null;
  sandbox?: SandboxMode | null;
  config?:
    | {
        [key in string]?: JsonValue;
      }
    | null;
  developerInstructions?: string | null;
  ephemeral?: boolean | null;
  environments?: [];
  dynamicTools?: Array<DynamicToolSpec> | null;
};

export type TurnStartParams = {
  threadId: string;
  clientUserMessageId?: string | null;
  input: Array<UserInput>;
  environments?: [];
  approvalPolicy?: AskForApproval | null;
  sandboxPolicy?: SandboxPolicy | null;
  model?: string | null;
  effort?: ReasoningEffort | null;
};

export type UserInput =
  | {
      type: 'text';
      text: string;
      text_elements: Array<TextElement>;
    }
  | {
      type: 'image';
      detail?: ImageDetail;
      url: string;
    }
  | {
      type: 'localImage';
      detail?: ImageDetail;
      path: string;
    }
  | {
      type: 'audio';
      url: string;
    }
  | {
      type: 'localAudio';
      path: string;
    }
  | {
      type: 'skill';
      name: string;
      path: string;
    }
  | {
      type: 'mention';
      name: string;
      path: string;
    };
