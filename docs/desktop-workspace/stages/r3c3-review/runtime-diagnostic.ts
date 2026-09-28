// Bounded protocol diagnosis; prints only thread/start outcome, never account or input payloads.
import { spawn } from "node:child_process";
import { createInterface } from "node:readline";
import { mkdir } from "node:fs/promises";
import { resolve, join } from "node:path";
import {
  discussionPolicy,
  sharedInstructions,
} from "../../../../app/desktop/src/main/codex/policy";
import { sharedToolDefinitions } from "../../../../app/desktop/src/main/shared-context/catalog";
function normalize(value: any): any {
  if (Array.isArray(value)) return value.map(normalize);
  if (!value || typeof value !== "object") return value;
  const result = Object.fromEntries(
    Object.entries(value).map(([k, v]) => [k, normalize(v)]),
  ) as any;
  if (Array.isArray(result.items)) {
    result.items = result.items[0];
    delete result.additionalItems;
  }
  return result;
}
async function main() {
  const profile = process.argv[2]!;
  const cwd = join(profile, "codex/diagnostic");
  await mkdir(cwd, { recursive: true });
  const child = spawn(
    resolve(
      "node_modules/@openai/codex-darwin-arm64/vendor/aarch64-apple-darwin/bin/codex",
    ),
    [
      "app-server",
      "--listen",
      "stdio://",
      ...Object.entries(discussionPolicy).flatMap(([k, v]) => [
        "-c",
        k + "=" + JSON.stringify(v),
      ]),
    ],
    {
      cwd,
      env: {
        HOME: process.env.HOME,
        PATH: process.env.PATH,
        TMPDIR: process.env.TMPDIR,
        LANG: "en_US.UTF-8",
        CODEX_HOME: join(profile, "codex/home"),
      },
      stdio: "pipe",
    },
  );
  child.stderr.resume();
  let sequence = 0;
  const pending = new Map<number, (v: any) => void>();
  createInterface({ input: child.stdout }).on("line", (line) => {
    const value = JSON.parse(line);
    pending.get(value.id)?.(value);
  });
  const request = (method: string, params: unknown) =>
    new Promise<any>((resolve) => {
      const id = ++sequence;
      pending.set(id, resolve);
      child.stdin.write(JSON.stringify({ id, method, params }) + "\n");
    });
  const timer = setTimeout(() => child.kill(), 20000);
  try {
    await request("initialize", {
      clientInfo: { name: "gobble_app", title: "Gobble", version: "0.1.0" },
      capabilities: { experimentalApi: true },
    });
    child.stdin.write(
      JSON.stringify({ method: "initialized", params: {} }) + "\n",
    );
    const result = await request("thread/start", {
      model: "gpt-6-astra",
      modelProvider: "openai",
      cwd,
      sandbox: "read-only",
      approvalPolicy: "never",
      config: { ...discussionPolicy, "features.code_mode.enabled": true },
      developerInstructions: sharedInstructions,
      environments: [],
      dynamicTools: sharedToolDefinitions.map((t) => ({
        type: "function",
        name: t.name,
        description: t.description,
        inputSchema: normalize(t.schema),
      })),
      ephemeral: true,
    });
    console.log(
      JSON.stringify(
        result.error
          ? { method: "thread/start", error: result.error }
          : { method: "thread/start", success: true },
      ),
    );
  } finally {
    clearTimeout(timer);
    child.kill();
  }
}
main().catch((e) => console.error(String(e)));
