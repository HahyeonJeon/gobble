# Project Workspace — Plan 01

> **Index:** [Plan](plan-index.md)
> **Source:** [Task hierarchy](tasks/tasks-index.md)

## Shared context

| Item | Contract |
|---|---|
| Work / purpose | Deliver the approved Project-centered collaboration proof |
| Output | Two real agent attachments, shared views/context, restore, engine query and local Mac package |
| Baseline | Gobble `7fe3d5c9e10d0cc104a15b6e7aa3def0d15b98f1` |
| Scope / acceptance | [Desktop review](../README.md) |
| Interface | [Workspace design](../../../.gobbi/projects/gobble/memory/design/feature/agent-workspace.md) |
| Architecture | [Project contract](../../../.gobbi/projects/gobble/memory/design/architecture/project-workspace-contract.md) |
| Target | Current macOS/arm64 host; engine containers retain linux/amd64 |
| Authority | User approved local implementation; present a sketch before each group and obtain approval after its tested result before the next group; no source data deletion, provider-token import, publication or automatic execution |
| Protected paths | Synced sources/ outside this clone; existing engine/assay contracts; user analysis inputs and runs |
| Grouping | Shared contracts, Go service, workspace, provider, contextual tools, integration evidence and packaging have distinct responsibility/dependency boundaries |

## Execution groups

Lower order numbers execute first. Requires edges are authoritative.
Shared paths have one writer at a time. Unexpected source or contract changes
return to the owning earlier group and update this plan before dependent work.

### `task-01-contracts` — 공통 계약과 프로세스 경계

| Attribute | Value |
|---|---|
| Order | 1 |
| Combined paths | 1.1, 1.2 |
| Requires | None |
| Accountable role | 앱 계약 설계·구현 담당 |
| Compatible skills | gobbi:electron-contract, gobbi:electron-design, gobbi:typescript-typing |

| Path | Work | Boundary | Output |
|---|---|---|---|
| 1.1 · 식별자와 메시지 | Project·Agent·surface·selection·decision 메시지를 버전 있는 계약으로 정의한다. | 전송 가능한 값과 오류; provider 및 engine 내부 타입을 복제하지 않음 | 공통 계약과 버전·오류 정의 |
| 1.2 · 프로세스와 상태 경계 | 프로세스별 책임과 좁은 preload, 상태 소유자 및 저장 형식을 구성한다. | 하나의 앱 인스턴스와 로컬 프로필 | 프로세스 골격과 소유권이 명확한 상태 계약 |

- **Why combined:** 클라이언트와 서비스가 함께 사용하는 경계를 하나의 일관된 결과로 확정한다.
- **Group outcome / verification:** 검증 가능한 공통 스키마와 분리된 main/preload/renderer 구성.
- **Stop:** 화면별 동작, provider 통신과 실행 제어 구현을 포함하지 않는다.
- **Constraints and authority:** Shared scope and protected paths apply; no silent provider/runtime fallback. Missing actual evidence stays open.
- **Design:** 프로젝트 계약의 Identity, Renderer to host, Persistence; governing documents linked above.
- **Writer frontier:** app/contracts/; app/desktop의 프로세스 진입·빌드 경계.
- **Handoffs:** Receive the required groups’ compatible contracts/results; pass the owned outcome to dependent groups. Return failures to their source owner.
- **Metadata:** Account/runtime/platform dependencies below apply where relevant.

### `task-02-project-service` — Project와 기존 실행의 서비스 연결

| Attribute | Value |
|---|---|
| Order | 2 |
| Combined paths | 2.1, 2.2 |
| Requires | `task-01-contracts` |
| Accountable role | Go 애플리케이션 서비스 담당 |
| Compatible skills | gobbi:go-development, gobbi:go-security |

| Path | Work | Boundary | Output |
|---|---|---|---|
| 2.1 · 프로젝트와 파일 | 사용자가 선택한 폴더를 등록하고 범위가 제한된 파일·CSV·이미지를 제공한다. | canonical root와 명시적 파일 제한; 원본 자동 수정 없음 | 내구성 있는 Project 등록 및 자원 조회 |
| 2.2 · 런타임과 실행 조회 | 기존 Run을 등록·발견하고 기록된 런타임의 Inspect/Monitor와 로그를 제공한다. | 이미지·daemon·workspace identity 보존; 조회와 관찰 시각 분리 | 호환 Run projection 및 명시적 unavailable 상태 |

- **Why combined:** 등록된 로컬 자원과 호환 엔진 조회는 같은 Project 범위와 서비스 권한을 사용한다.
- **Group outcome / verification:** Project를 등록하고 파일 및 호환 가능한 기존 Run을 조회하는 로컬 서비스.
- **Stop:** 새 분석 실행과 Start/Stop/Resume 명령을 노출하지 않는다.
- **Constraints and authority:** Shared scope and protected paths apply; no silent provider/runtime fallback. Missing actual evidence stays open.
- **Design:** 프로젝트 계약의 Host to Go service, Runtime routing; governing documents linked above.
- **Writer frontier:** cmd/gobble-service/; internal/appservice/; app/contracts의 Go 경계.
- **Handoffs:** Receive the required groups’ compatible contracts/results; pass the owned outcome to dependent groups. Return failures to their source owner.
- **Metadata:** Account/runtime/platform dependencies below apply where relevant.

### `task-03-workspace` — Project 공유 작업영역

| Attribute | Value |
|---|---|
| Order | 3 |
| Combined paths | 3.1, 3.2, 3.3 |
| Requires | `task-01-contracts`, `task-02-project-service` |
| Accountable role | Desktop UI 및 작업공간 담당 |
| Compatible skills | gobbi:desktop-architecture, gobbi:desktop-interface, gobbi:electron-development, gobbi:react-development |

| Path | Work | Boundary | Output |
|---|---|---|---|
| 3.1 · 공유 화면 제어 | 탭·1/2 패널·고정·분할·닫기·중복 요청·Project 전환을 단일 controller에서 처리한다. | Project 소유 화면과 명시적 렌더 준비 상태 | 일관된 화면·배치 상태와 조작 결과 |
| 3.2 · 표시와 접근 | 파일·CSV·이미지·Run·로그 화면 및 오른쪽 단일 입력창 채팅을 키보드와 포인터로 사용 가능하게 만든다. | 승인된 중앙 작업영역과 반응형 한/두 패널 | 직접 조작 가능한 접근성 있는 화면 |
| 3.3 · 작업공간 복원 | Project·Agent 참조·패널·선택·대기 질문 및 안전한 창 상태를 저장·복원한다. | 메시지·실행의 자동 재전송 없음 | 재시작·누락 자원에 대응하는 복원 동작 |

- **Why combined:** 패널 제어와 실제 UI는 동일한 Project 상태·배치 책임을 구현한다.
- **Group outcome / verification:** Project 탐색, 중앙 상하 패널과 오른쪽 채팅 영역을 직접 조작하고 복원할 수 있는 앱.
- **Stop:** 실제 Agent 호출과 화면 관찰 도구 연결을 포함하지 않는다.
- **Constraints and authority:** Shared scope and protected paths apply; no silent provider/runtime fallback. Missing actual evidence stays open.
- **Design:** 작업공간 설계의 View hierarchy, Interaction contract, Desktop lifetime; governing documents linked above.
- **Writer frontier:** app/desktop의 workspace controller, persistence, preload, renderer 및 연관 스타일.
- **Handoffs:** Receive the required groups’ compatible contracts/results; pass the owned outcome to dependent groups. Return failures to their source owner.
- **Metadata:** Account/runtime/platform dependencies below apply where relevant.

### `task-04-agents` — 공식 계정과 여러 Agent 연결

| Attribute | Value |
|---|---|
| Order | 4 |
| Combined paths | 4.1, 4.2 |
| Requires | `task-01-contracts`, `task-03-workspace` |
| Accountable role | Provider 연동 담당 |
| Compatible skills | openai-docs, gobbi:electron-development, gobbi:typescript-async |

| Path | Work | Boundary | Output |
|---|---|---|---|
| 4.1 · 공식 runtime과 로그인 | 고정 호환 Codex 실행 파일과 공식 ChatGPT 로그인·모델 목록을 연결한다. | 앱 전용 계정 저장; 사용자가 browser 로그인 수행 | 명시적 계정·runtime 상태와 모델 선택 |
| 4.2 · 독립 Agent 대화 | 두 peer attachment의 thread·turn·수신자·중단·재연결·제출 불확실성을 분리한다. | Project 공유 기록과 각 provider 대화의 권한 구분 | 지정된 Agent의 대화와 다른 Agent를 보존하는 중단 |

- **Why combined:** 계정·런타임 생명주기와 각 attachment의 대화 상태가 같은 provider 경계에 속한다.
- **Group outcome / verification:** 같은 Project에 두 실제 Agent를 연결하고 수신자를 선택해 대화할 수 있다.
- **Stop:** 소스 쓰기, 분석 실행, 자동 broadcast와 자동 Agent 위임을 포함하지 않는다.
- **Constraints and authority:** Shared scope and protected paths apply; no silent provider/runtime fallback. Missing actual evidence stays open.
- **Design:** 프로젝트 계약의 Host to provider and tools, Multi-agent coordination; governing documents linked above.
- **Writer frontier:** app/desktop의 provider adapter, agent bindings, account UI 및 대화 전달.
- **Handoffs:** Receive the required groups’ compatible contracts/results; pass the owned outcome to dependent groups. Return failures to their source owner.
- **Metadata:** Account/runtime/platform dependencies below apply where relevant.

### `task-05-shared-context` — Agent와 사용자의 함께 보기

| Attribute | Value |
|---|---|
| Order | 5 |
| Combined paths | 5.1, 5.2 |
| Requires | `task-02-project-service`, `task-03-workspace`, `task-04-agents` |
| Accountable role | 공유 맥락 및 도구 통합 담당 |
| Compatible skills | openai-docs, gobbi:electron-development, gobbi:react-development, gobbi:typescript-async |

| Path | Work | Boundary | Output |
|---|---|---|---|
| 5.1 · 열기와 실제 관찰 | Project/Agent 범위의 도구로 화면 열기·배치·관찰·임시 화면 해제를 연결한다. | 실제 렌더 acknowledgment; 없는 화면을 관찰했다고 기록하지 않음 | 실제 상태와 한정된 이미지 도구 결과 |
| 5.2 · 선택 전달과 질문 | 행·텍스트·이미지 영역을 버전과 수신자에 묶고 근거 기반 질문·응답을 보존한다. | 오래된 근거 무효화; 닫기·침묵은 답변이 아님 | 명시적 context 전달 및 내구성 있는 질문·응답 |

- **Why combined:** 화면 관찰·선택 전송·질문 응답은 같은 근거 identity와 실제 전달 결과를 공유한다.
- **Group outcome / verification:** Agent가 두 화면을 열고 실제 관찰을 사용하며, 사용자가 선택한 근거를 다른 Agent에게 전달하고 응답받는다.
- **Stop:** 임의 JS·OS 조작·무제한 캡처·소스 변경·실행 명령을 포함하지 않는다.
- **Constraints and authority:** Shared scope and protected paths apply; no silent provider/runtime fallback. Missing actual evidence stays open.
- **Design:** 작업공간 설계의 Pointing and context sharing, Questions and decisions; governing documents linked above.
- **UI revision (2026-09-07):** 중앙 Pane 상하 배치, 우측 Project 채팅, 일반 메시지와 질문 답변에 공통 입력창 하나를 사용한다. 별도 Questions 화면과 답변 폼은 만들지 않는다. v2 배치·draft/reply 상태, renderer/chat 구성과 관련 테스트가 이 그룹의 선행 작업이다.
- **Writer frontier:** app/desktop의 scoped MCP bridge, host broker, observations, selections, decisions. The [stage 5 design proposal](../stages/05-shared-context.md) requests an explicit dynamic-tool transport change; adopt it only after user review.
- **Handoffs:** Receive the required groups’ compatible contracts/results; pass the owned outcome to dependent groups. Return failures to their source owner.
- **Metadata:** Account/runtime/platform dependencies below apply where relevant.

### `task-06-integration-evidence` — 실제 통합과 복원 근거

| Attribute | Value |
|---|---|
| Order | 6 |
| Combined paths | 6.1, 6.2 |
| Requires | `task-02-project-service`, `task-03-workspace`, `task-04-agents`, `task-05-shared-context` |
| Accountable role | 통합 검증 담당 |
| Compatible skills | gobbi:electron-testing, gobbi:react-testing, gobbi:go-testing |

| Path | Work | Boundary | Output |
|---|---|---|---|
| 6.1 · 동작·오류·복원 | Project 격리, 반복 요청, 늦은 결과, 질문 무효화와 재시작의 완료 여부를 판정한다. | 계약과 승인된 UX; 실패 은폐 없음 | 관찰 가능한 동작 및 회복 근거 |
| 6.2 · 실제 엔진·Agent 연결 | CLI 실행과 앱 표시, 실제 Agent의 관찰·이미지 소비, 앱 종료 후 실행 독립성을 판정한다. | 실제 Docker와 사용자 로그인 필요; mock을 실제 증거로 대체하지 않음 | 실제 통합 근거 또는 정확한 남은 외부 조건 |

- **Why combined:** 동일한 사용자 여정에 대한 데이터·provider·창 생명주기 근거를 한 범위에서 판정한다.
- **Group outcome / verification:** 구현 주장마다 실제·대체·미확인 결과를 구분한 통합 근거.
- **Stop:** 검증 실패 원인의 제품 정책을 임의 변경하거나 일반 배포를 승인하지 않는다.
- **Constraints and authority:** Shared scope and protected paths apply; no silent provider/runtime fallback. Missing actual evidence stays open.
- **Design:** 작업공간 storyboard 및 검토 문서의 완료 판정; governing documents linked above.
- **Writer frontier:** 관련 테스트와 fixture; docs/desktop-workspace의 검증 결과; 격리된 검증 출력.
- **Handoffs:** Receive the required groups’ compatible contracts/results; pass the owned outcome to dependent groups. Return failures to their source owner.
- **Metadata:** Account/runtime/platform dependencies below apply where relevant.

### `task-07-native-package` — 로컬 Mac 앱 패키지와 인수

| Attribute | Value |
|---|---|
| Order | 7 |
| Combined paths | 7.1, 7.2 |
| Requires | `task-06-integration-evidence` |
| Accountable role | Desktop 패키징 담당 |
| Compatible skills | gobbi:electron-packaging, gobbi:desktop-macos |

| Path | Work | Boundary | Output |
|---|---|---|---|
| 7.1 · 실행 가능한 로컬 패키지 | 프로세스 실행 파일과 프론트엔드를 포함한 로컬 .app을 준비한다. | 고정 runtime의 출처·digest·notice; 다른 앱의 내부 설치 경로 의존 없음 | 로컬 앱 artifact와 버전 정보 |
| 7.2 · 패키지 사용 확인 | 패키지에서 Project·로그인·파일 경로·공유 보기·복원 결과와 제한을 정리한다. | 현재 macOS 대상; sign-in과 Docker 외부 조건 명시 | 사용 방법과 artifact 범위의 인수 근거 |

- **Why combined:** 필수 실행 파일의 배치와 설치된 앱의 열기·경로·연결 동작이 하나의 artifact 결과를 이룬다.
- **Group outcome / verification:** 현재 Mac에서 열 수 있는 로컬 .app과 동봉 runtime·사용·제약 설명.
- **Stop:** 서명·공증·공개 배포·자동 업데이트·다른 OS 지원을 선언하지 않는다.
- **Constraints and authority:** Shared scope and protected paths apply; no silent provider/runtime fallback. Missing actual evidence stays open.
- **Design:** 프로젝트 계약의 프로세스 topology와 provider acquisition; 검토 문서의 초기 범위; governing documents linked above.
- **Writer frontier:** app/desktop packaging 설정·resources·관련 문서; 로컬 패키지 출력.
- **Handoffs:** Receive the required groups’ compatible contracts/results; pass the owned outcome to dependent groups. Return failures to their source owner.
- **Metadata:** Account/runtime/platform dependencies below apply where relevant.

## Unresolved factual metadata

| Item | Evidence | Execution effect | Groups |
|---|---|---|---|
| Docker daemon | Not reachable on this host on 2026-09-06 | Service/UI work can proceed; actual engine integration requires Docker readiness | 02, 06, 07 |
| Account login | User completed app-specific browser login; stage 4 live replies, isolated interrupt and restart were verified | Stage 5 tool/image consumption still needs its own live evidence; no token import | 04, 05, 06, 07 |
| Provider capabilities | Stage 4 verified official Codex 0.153.4 login, two live peer responses, isolated interrupt and normal restart alongside automated checks | Thread-scoped MCP/image delivery remain stage 5/6, with no fake fallback | 04, 05, 06 |
| Exact frontend pins | Exact pins and lockfile are owned by app/package.json; implementation evidence is in stage records | Changes must satisfy build/test peer ranges and reopen affected runtime evidence | 01, 03, 07 |
| Native distribution | No Gobble .app exists yet | Package scope is current-host local artifact, not signed/public release | 07 |

## Stage 5 sequential review checkpoints — 2026-09-07

User approved the revised layout and requested User/Agent communication tools.
[Implementation slice 5.1](../stages/05-1-chat-foundation.md) covers right chat, stacked Panes,
v2 migration and the common Selection definition. This slice numbering follows the Stage 5
design's verification gates and does not replace the stable task hierarchy paths above.
Pointing/overlays follow in slice 5.2; immutable addressed evidence in 5.3; question replies in
5.4; live qualification in 5.5. Present each slice for review before implementing the next.
