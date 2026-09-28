# PDF contract sketch — non-executable proposal

These shapes illustrate responsibility and identity. They are not exported contracts, accepted schema versions or PDF.js API types. Production uses the project's closed runtime schemas and requires a version/migration review.

```typescript
type PdfPageIdentity = {
  projectId: ProjectId;
  resource: { kind: 'file'; resourceId: ResourceId };
  dataRevision: Sha256;             // exact source bytes
  pageIndex: number;                // zero-based physical page
};

type PdfPageModel = {
  identity: PdfPageIdentity;
  decoderProfile: string;          // pinned adapter/extraction/render policy
  pageModelHash: Sha256;
  viewBox: [number, number, number, number];
  userUnit: number;
  intrinsicRotation: 0 | 90 | 180 | 270;
  textCapability:
    | { kind: 'exact'; items: PdfTextItem[] }
    | { kind: 'unavailable'; reason: 'no-text' | 'ambiguous' | 'limit' };
};

type PdfTextRange = {
  itemId: string;                   // from the retained page model
  startUtf16: number;               // inclusive; no split surrogate pair
  endUtf16: number;                 // exclusive
};

type PdfSelector =
  | { kind: 'pdf-page' }
  | {
      kind: 'pdf-region';
      coordinateSpace: 'pdf-page-user-space';
      rect: [number, number, number, number]; // min x/y, max x/y
    }
  | { kind: 'pdf-text'; ranges: PdfTextRange[] };

type PdfTarget = {
  page: PdfPageIdentity;
  decoderProfile: string;
  pageModelHash: Sha256;
  selector: PdfSelector;
};

// Main owns job/source binding. No arbitrary URL, path, caller quote or image.
interface PdfDecoder {
  open(bytes: Uint8Array, profile: DecoderProfile, signal: AbortSignal): Promise<Session>;
  page(session: Session, pageIndex: number, signal: AbortSignal): Promise<PageResult>;
  capture(session: Session, target: PdfTarget, signal: AbortSignal): Promise<CaptureResult>;
  close(session: Session): Promise<void>;
}
```

The public target's Project/Resource shape must reuse the actual existing ResourceRef schema rather than introduce a parallel identity type. Field names above describe intent; the final version is assigned at R3c1/R3c2 after inspecting all consumers. Session and job handles are ephemeral and never appear in persisted targets.

Validation belongs to Main plus pure contract/geometry functions. It checks current Project and Surface binding, page index bounds, exact source/profile/model identity, live render receipt, finite geometry, rectangle containment, nonempty ordered text ranges and valid Unicode boundaries. Decoder replies are untrusted protocol input and cannot authorize file reads or publication.

Quote text and highlight quads are derived evidence from an accepted model, not point-command authority. Render scale, raster size, crop transform and actual image byte count belong to capture representation metadata. No fabricated crop/pixel data is accepted from React.

An Agent observation binds a returned page model and visible area to a read ID. A text pointer is a subset of returned item/ranges. A region pointer is contained in the delivered image's mapped page area. If image delivery is unavailable, metadata alone does not grant region pointing authority.
