import { Type, type Static } from '@sinclair/typebox';
import {
  closed,
  ProjectIdSchema,
  ResourceIdSchema,
  RevisionSchema,
  SurfaceIdSchema,
} from './identity';
import { PageSchema, RectSchema, MatrixSchema, PROFILE } from './pdf-decoder';
import { assertRegion, pixelRegion } from './pdf-geometry';

export const PdfSourceSchema = Type.Object(
  {
    kind: Type.Literal('pdf'),
    base64: Type.String({ maxLength: 11_184_812, pattern: '^[A-Za-z0-9+/]+={0,2}$' }),
  },
  closed,
);
export const PdfContentSchema = Type.Object(
  {
    kind: Type.Literal('pdf'),
    pageCount: Type.Integer({ minimum: 1, maximum: 200 }),
    page: PageSchema,
  },
  closed,
);
export const PdfNavigationSchema = Type.Object(
  {
    fitWidth: Type.Optional(Type.Boolean()),
    pageIndex: Type.Integer({ minimum: 0, maximum: 199 }),
    scale: Type.Union([Type.Literal(0.75), Type.Literal(1), Type.Literal(1.25), Type.Literal(1.5)]),
    rotation: Type.Union([Type.Literal(0), Type.Literal(90), Type.Literal(180), Type.Literal(270)]),
  },
  closed,
);
export const defaultPdfNavigation = (): PdfNavigation => ({ pageIndex: 0, scale: 1, rotation: 0 });
export const PdfSelectionSchema = Type.Object(
  {
    kind: Type.Literal('pdf'),
    coordinateSpace: Type.Literal('pdf-user-space'),
    scope: Type.Union([Type.Literal('page'), Type.Literal('region')]),
    profile: Type.Literal(PROFILE),
    pageIndex: Type.Integer({ minimum: 0, maximum: 199 }),
    modelHash: RevisionSchema,
    region: RectSchema,
  },
  closed,
);
export const PdfTargetSchema = Type.Object(
  {
    schemaVersion: Type.Literal(5),
    projectId: ProjectIdSchema,
    resource: Type.Object({ kind: Type.Literal('file'), resourceId: ResourceIdSchema }, closed),
    dataRevision: RevisionSchema,
    origin: Type.Optional(Type.Object({ surfaceId: SurfaceIdSchema }, closed)),
    selection: PdfSelectionSchema,
  },
  closed,
);
/** Geometry accompanies the immutable PNG; a rendition ID never becomes durable authority. */
export const PdfCaptureGeometrySchema = Type.Object(
  {
    profile: Type.Literal(PROFILE),
    pageIndex: Type.Integer({ minimum: 0, maximum: 199 }),
    modelHash: RevisionSchema,
    viewBox: RectSchema,
    userUnit: Type.Number({ minimum: 0.001, maximum: 75000 }),
    intrinsicRotation: Type.Union([
      Type.Literal(0),
      Type.Literal(90),
      Type.Literal(180),
      Type.Literal(270),
    ]),
    viewport: MatrixSchema,
  },
  closed,
);
export const PdfImageRepresentationSchema = Type.Object(
  {
    kind: Type.Literal('image'),
    width: Type.Integer({ minimum: 1, maximum: 1536 }),
    height: Type.Integer({ minimum: 1, maximum: 1536 }),
    originalWidth: Type.Integer({ minimum: 1, maximum: 4_000_000 }),
    originalHeight: Type.Integer({ minimum: 1, maximum: 4_000_000 }),
    crop: Type.Object(
      {
        x: Type.Integer({ minimum: 0 }),
        y: Type.Integer({ minimum: 0 }),
        width: Type.Integer({ minimum: 1 }),
        height: Type.Integer({ minimum: 1 }),
      },
      closed,
    ),
    pdf: PdfCaptureGeometrySchema,
  },
  closed,
);
export type PdfContent = Static<typeof PdfContentSchema>;
export type PdfNavigation = Static<typeof PdfNavigationSchema>;
export type PdfSelection = Static<typeof PdfSelectionSchema>;
export type PdfTarget = Static<typeof PdfTargetSchema>;
export type PdfImageRepresentation = Static<typeof PdfImageRepresentationSchema>;

export function validatePdfSelection(selection: PdfSelection, content?: PdfContent): void {
  assertRegion(selection.region, content?.page.model.viewBox ?? selection.region);
  if (
    content &&
    (selection.profile !== content.page.model.profile ||
      selection.modelHash !== content.page.modelHash ||
      selection.pageIndex !== content.page.model.pageIndex)
  )
    throw new Error('PDF selection does not match this page model.');
  if (
    content &&
    selection.scope === 'page' &&
    JSON.stringify(selection.region) !== JSON.stringify(content.page.model.viewBox)
  )
    throw new Error('Whole-page selection must cover the page box.');
}
export function validatePdfImage(target: PdfTarget, view: PdfImageRepresentation): void {
  validatePdfSelection(target.selection);
  assertRegion(target.selection.region, view.pdf.viewBox);
  if (
    target.selection.profile !== view.pdf.profile ||
    target.selection.modelHash !== view.pdf.modelHash ||
    target.selection.pageIndex !== view.pdf.pageIndex
  )
    throw new Error('PDF capture model mismatch.');
  if (
    target.selection.scope === 'page' &&
    JSON.stringify(target.selection.region) !== JSON.stringify(view.pdf.viewBox)
  )
    throw new Error('PDF page capture is incomplete.');
  if (
    view.originalWidth * view.originalHeight > 4_000_000 ||
    view.width !== view.crop.width ||
    view.height !== view.crop.height
  )
    throw new Error('PDF capture cannot be resized.');
  const expected = pixelRegion(
    target.selection.region,
    view.pdf.viewport,
    view.originalWidth,
    view.originalHeight,
  );
  if (JSON.stringify(expected) !== JSON.stringify(view.crop))
    throw new Error('PDF capture crop mismatch.');
}
