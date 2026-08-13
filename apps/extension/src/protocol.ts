export const PROTOCOL_VERSION = '1.0' as const;

export interface ContentPacket {
  protocolVersion: typeof PROTOCOL_VERSION;
  captureId: string;
  capturedAt: string;
  source: {
    url: string;
    title: string;
    site?: string;
    author?: string;
    published?: string;
    language?: string;
    description?: string;
  };
  content: {
    markdown: string;
    html?: string;
  };
  selection?: {
    markdown?: string;
    html?: string;
  };
  highlights?: Record<string, unknown>[];
  metadata?: {
    wordCount?: number;
    image?: string;
    favicon?: string;
    schemaOrg?: unknown;
    metaTags?: Array<{ name?: string | null; property?: string | null; content: string | null }>;
    variables?: Record<string, unknown>;
    [key: string]: unknown;
  };
  media?: {
    transcript?: string;
    [key: string]: unknown;
  };
}

export interface SubmitResult {
  captureId: string;
  notePath?: string;
  aiStatus: 'ok' | 'failed' | 'disabled' | 'unknown' | 'pending';
  duplicate?: boolean;
}
