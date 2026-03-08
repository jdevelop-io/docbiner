'use client';

import { useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save, Eye, Loader2 } from 'lucide-react';
import { api } from '@/lib/api';
import type { Template } from '@/lib/types';
import { TemplateEditor } from '@/components/template-editor';
import { cn } from '@/lib/utils';

const DEFAULT_HTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>{{title}}</title>
</head>
<body>
  <h1>{{title}}</h1>
  <p>{{content}}</p>
</body>
</html>`;

const DEFAULT_SAMPLE_DATA = JSON.stringify(
  { title: 'Hello World', content: 'This is a sample document.' },
  null,
  2,
);

export default function TemplateEditorPage() {
  const params = useParams();
  const templateId = params.id as string;
  const isNew = templateId === 'new';

  // Fetch existing template
  const { data: template, isLoading: isLoadingTemplate, dataUpdatedAt } = useQuery({
    queryKey: ['template', templateId],
    queryFn: () => api.get<Template>(`/v1/templates/${templateId}`),
    enabled: !isNew,
  });

  if (!isNew && isLoadingTemplate) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  // key forces remount when template data loads, so initial state is correct
  return (
    <TemplateForm
      key={dataUpdatedAt}
      templateId={templateId}
      isNew={isNew}
      template={template ?? null}
    />
  );
}

function TemplateForm({
  templateId,
  isNew,
  template,
}: {
  templateId: string;
  isNew: boolean;
  template: Template | null;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();

  const [name, setName] = useState(template?.name ?? '');
  const [engine, setEngine] = useState<'handlebars' | 'liquid'>(template?.engine ?? 'handlebars');
  const [htmlContent, setHtmlContent] = useState(template?.html_content ?? DEFAULT_HTML);
  const [cssContent, setCssContent] = useState(template?.css_content ?? '');
  const [sampleData, setSampleData] = useState(
    template?.sample_data
      ? JSON.stringify(template.sample_data, null, 2)
      : DEFAULT_SAMPLE_DATA,
  );
  const [previewHtml, setPreviewHtml] = useState<string | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: async () => {
      let parsedSampleData: Record<string, unknown> = {};
      try {
        parsedSampleData = JSON.parse(sampleData);
      } catch {
        // Keep empty object if invalid JSON
      }

      const body = {
        name,
        engine,
        html_content: htmlContent,
        css_content: cssContent || undefined,
        sample_data: parsedSampleData,
      };

      if (isNew) {
        return api.post<Template>('/v1/templates', body);
      }
      return api.put<Template>(`/v1/templates/${templateId}`, body);
    },
    onSuccess: (data) => {
      setSaveError(null);
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      queryClient.invalidateQueries({ queryKey: ['template', templateId] });
      if (isNew && data?.id) {
        router.replace(`/templates/${data.id}`);
      }
    },
    onError: (err: Error) => {
      setSaveError(err.message);
    },
  });

  // Preview mutation
  const previewMutation = useMutation({
    mutationFn: async () => {
      let parsedSampleData: Record<string, unknown> = {};
      try {
        parsedSampleData = JSON.parse(sampleData);
      } catch {
        throw new Error('Invalid JSON in sample data');
      }

      if (isNew) {
        // For new templates, we need to save first, then preview
        // Or use a generic preview endpoint
        return api.post<{ html: string }>('/v1/templates/preview', {
          engine,
          html_content: htmlContent,
          css_content: cssContent || undefined,
          data: parsedSampleData,
        });
      }

      return api.post<{ html: string }>(`/v1/templates/${templateId}/preview`, {
        data: parsedSampleData,
      });
    },
    onSuccess: (data) => {
      setPreviewHtml(data.html);
      setPreviewError(null);
    },
    onError: (err: Error) => {
      setPreviewError(err.message);
      setPreviewHtml(null);
    },
  });

  const handleSave = useCallback(() => {
    if (!name.trim()) {
      setSaveError('Template name is required');
      return;
    }
    saveMutation.mutate();
  }, [name, saveMutation]);

  const handlePreview = useCallback(() => {
    previewMutation.mutate();
  }, [previewMutation]);

  return (
    <div className="flex h-[calc(100vh-theme(spacing.14)-theme(spacing.12))] flex-col lg:h-[calc(100vh-theme(spacing.12))]">
      {/* Top bar */}
      <div className="flex shrink-0 flex-wrap items-center gap-3 pb-4">
        <button
          onClick={() => router.push('/templates')}
          className="rounded-md p-1.5 text-muted-foreground hover:text-foreground transition-colors"
          title="Back to templates"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>

        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="Template name..."
          className="min-w-0 flex-1 rounded-md border border-input bg-background px-3 py-1.5 text-sm font-medium shadow-sm transition-colors placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
        />

        <select
          value={engine}
          onChange={(e) => setEngine(e.target.value as 'handlebars' | 'liquid')}
          className="rounded-md border border-input bg-background px-3 py-1.5 text-sm shadow-sm transition-colors focus:outline-none focus:ring-2 focus:ring-ring"
        >
          <option value="handlebars">Handlebars</option>
          <option value="liquid">Liquid</option>
        </select>

        <button
          onClick={handlePreview}
          disabled={previewMutation.isPending}
          className="inline-flex items-center gap-2 rounded-md border border-input bg-background px-3 py-1.5 text-sm font-medium shadow-sm transition-colors hover:bg-accent hover:text-accent-foreground disabled:opacity-50"
        >
          {previewMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Eye className="h-4 w-4" />
          )}
          Preview
        </button>

        <button
          onClick={handleSave}
          disabled={saveMutation.isPending}
          className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-1.5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
        >
          {saveMutation.isPending ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Save className="h-4 w-4" />
          )}
          Save
        </button>
      </div>

      {/* Error messages */}
      {saveError && (
        <div className="mb-3 shrink-0 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-900/10 dark:text-red-400">
          {saveError}
        </div>
      )}

      {saveMutation.isSuccess && (
        <div className="mb-3 shrink-0 rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700 dark:border-green-900/50 dark:bg-green-900/10 dark:text-green-400">
          Template saved successfully.
        </div>
      )}

      {/* Split view: editor + preview */}
      <div className="flex min-h-0 flex-1 gap-4">
        {/* Editor */}
        <div className={cn('min-h-0 flex-1', previewHtml !== null ? 'w-1/2' : 'w-full')}>
          <TemplateEditor
            htmlContent={htmlContent}
            cssContent={cssContent}
            sampleData={sampleData}
            onHtmlChange={setHtmlContent}
            onCssChange={setCssContent}
            onSampleDataChange={setSampleData}
          />
        </div>

        {/* Preview panel */}
        {previewHtml !== null && (
          <div className="flex w-1/2 min-h-0 flex-col rounded-lg border border-border bg-card">
            <div className="flex items-center justify-between border-b border-border px-4 py-2">
              <span className="text-sm font-medium">Preview</span>
              <button
                onClick={() => {
                  setPreviewHtml(null);
                  setPreviewError(null);
                }}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Close
              </button>
            </div>
            <div className="flex-1 min-h-0 overflow-auto bg-white">
              <iframe
                srcDoc={previewHtml}
                title="Template preview"
                className="h-full w-full border-0"
                sandbox="allow-same-origin"
              />
            </div>
          </div>
        )}

        {/* Preview error */}
        {previewError && previewHtml === null && (
          <div className="flex w-1/2 min-h-0 flex-col rounded-lg border border-border bg-card">
            <div className="flex items-center justify-between border-b border-border px-4 py-2">
              <span className="text-sm font-medium">Preview</span>
              <button
                onClick={() => setPreviewError(null)}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Close
              </button>
            </div>
            <div className="flex flex-1 items-center justify-center p-4">
              <p className="text-sm text-red-600 dark:text-red-400">{previewError}</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
