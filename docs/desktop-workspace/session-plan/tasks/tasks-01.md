# Project Workspace — Task Hierarchy 01

> **Index:** [Tasks](tasks-index.md)
> **Result:** [Plan](../plan-index.md)

## Work summary

One Project with two actual agent attachments, shared real-data views, scoped
pointing and questions, safe restoration, existing-engine queries and a local
Mac app package. Source mutation, new analysis controls, detached OS windows
and public distribution are outside this slice. Design and acceptance are
defined by the [review](../../README.md). All hierarchy branches below contribute
to this one outcome; numeric paths express decomposition only.

## 1 — 공통 계약과 프로세스 경계

Type: Group. 검증 가능한 공통 스키마와 분리된 main/preload/renderer 구성.

| Path | Leaf work | Boundary | Output |
|---|---|---|---|
| 1.1 · 식별자와 메시지 | Project·Agent·surface·selection·decision 메시지를 버전 있는 계약으로 정의한다. | 전송 가능한 값과 오류; provider 및 engine 내부 타입을 복제하지 않음 | 공통 계약과 버전·오류 정의 |
| 1.2 · 프로세스와 상태 경계 | 프로세스별 책임과 좁은 preload, 상태 소유자 및 저장 형식을 구성한다. | 하나의 앱 인스턴스와 로컬 프로필 | 프로세스 골격과 소유권이 명확한 상태 계약 |

## 2 — Project와 기존 실행의 서비스 연결

Type: Group. Project를 등록하고 파일 및 호환 가능한 기존 Run을 조회하는 로컬 서비스.

| Path | Leaf work | Boundary | Output |
|---|---|---|---|
| 2.1 · 프로젝트와 파일 | 사용자가 선택한 폴더를 등록하고 범위가 제한된 파일·CSV·이미지를 제공한다. | canonical root와 명시적 파일 제한; 원본 자동 수정 없음 | 내구성 있는 Project 등록 및 자원 조회 |
| 2.2 · 런타임과 실행 조회 | 기존 Run을 등록·발견하고 기록된 런타임의 Inspect/Monitor와 로그를 제공한다. | 이미지·daemon·workspace identity 보존; 조회와 관찰 시각 분리 | 호환 Run projection 및 명시적 unavailable 상태 |

## 3 — Project 공유 작업영역

Type: Group. Project 탐색, 중앙 상하 패널과 오른쪽 채팅 영역을 직접 조작하고 복원할 수 있는 앱.

| Path | Leaf work | Boundary | Output |
|---|---|---|---|
| 3.1 · 공유 화면 제어 | 탭·1/2 패널·고정·분할·닫기·중복 요청·Project 전환을 단일 controller에서 처리한다. | Project 소유 화면과 명시적 렌더 준비 상태 | 일관된 화면·배치 상태와 조작 결과 |
| 3.2 · 표시와 접근 | 파일·CSV·이미지·Run·로그 화면 및 오른쪽 단일 입력창 채팅을 키보드와 포인터로 사용 가능하게 만든다. | 승인된 중앙 작업영역과 반응형 한/두 패널 | 직접 조작 가능한 접근성 있는 화면 |
| 3.3 · 작업공간 복원 | Project·Agent 참조·패널·선택·대기 질문 및 안전한 창 상태를 저장·복원한다. | 메시지·실행의 자동 재전송 없음 | 재시작·누락 자원에 대응하는 복원 동작 |

## 4 — 공식 계정과 여러 Agent 연결

Type: Group. 같은 Project에 두 실제 Agent를 연결하고 수신자를 선택해 대화할 수 있다.

| Path | Leaf work | Boundary | Output |
|---|---|---|---|
| 4.1 · 공식 runtime과 로그인 | 고정 호환 Codex 실행 파일과 공식 ChatGPT 로그인·모델 목록을 연결한다. | 앱 전용 계정 저장; 사용자가 browser 로그인 수행 | 명시적 계정·runtime 상태와 모델 선택 |
| 4.2 · 독립 Agent 대화 | 두 peer attachment의 thread·turn·수신자·중단·재연결·제출 불확실성을 분리한다. | Project 공유 기록과 각 provider 대화의 권한 구분 | 지정된 Agent의 대화와 다른 Agent를 보존하는 중단 |

## 5 — Agent와 사용자의 함께 보기

Type: Group. Agent가 두 화면을 열고 실제 관찰을 사용하며, 사용자가 선택한 근거를 다른 Agent에게 전달하고 응답받는다.

| Path | Leaf work | Boundary | Output |
|---|---|---|---|
| 5.1 · 열기와 실제 관찰 | Project/Agent 범위의 도구로 화면 열기·배치·관찰·임시 화면 해제를 연결한다. | 실제 렌더 acknowledgment; 없는 화면을 관찰했다고 기록하지 않음 | 실제 상태와 한정된 이미지 도구 결과 |
| 5.2 · 선택 전달과 질문 | 행·텍스트·이미지 영역을 버전과 수신자에 묶고 근거 기반 질문·응답을 보존한다. | 오래된 근거 무효화; 닫기·침묵은 답변이 아님 | 명시적 context 전달 및 내구성 있는 질문·응답 |

## 6 — 실제 통합과 복원 근거

Type: Group. 구현 주장마다 실제·대체·미확인 결과를 구분한 통합 근거.

| Path | Leaf work | Boundary | Output |
|---|---|---|---|
| 6.1 · 동작·오류·복원 | Project 격리, 반복 요청, 늦은 결과, 질문 무효화와 재시작의 완료 여부를 판정한다. | 계약과 승인된 UX; 실패 은폐 없음 | 관찰 가능한 동작 및 회복 근거 |
| 6.2 · 실제 엔진·Agent 연결 | CLI 실행과 앱 표시, 실제 Agent의 관찰·이미지 소비, 앱 종료 후 실행 독립성을 판정한다. | 실제 Docker와 사용자 로그인 필요; mock을 실제 증거로 대체하지 않음 | 실제 통합 근거 또는 정확한 남은 외부 조건 |

## 7 — 로컬 Mac 앱 패키지와 인수

Type: Group. 현재 Mac에서 열 수 있는 로컬 .app과 동봉 runtime·사용·제약 설명.

| Path | Leaf work | Boundary | Output |
|---|---|---|---|
| 7.1 · 실행 가능한 로컬 패키지 | 프로세스 실행 파일과 프론트엔드를 포함한 로컬 .app을 준비한다. | 고정 runtime의 출처·digest·notice; 다른 앱의 내부 설치 경로 의존 없음 | 로컬 앱 artifact와 버전 정보 |
| 7.2 · 패키지 사용 확인 | 패키지에서 Project·로그인·파일 경로·공유 보기·복원 결과와 제한을 정리한다. | 현재 macOS 대상; sign-in과 Docker 외부 조건 명시 | 사용 방법과 artifact 범위의 인수 근거 |
