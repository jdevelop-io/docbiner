'use client';

import Editor, { type OnMount } from '@monaco-editor/react';
import { useCallback, useRef } from 'react';
import { Loader2 } from 'lucide-react';
import type { editor } from 'monaco-editor';

const DEFAULT_HTML = `<!DOCTYPE html>
<html>
<head>
  <style>
    body { font-family: 'Helvetica', sans-serif; padding: 40px; }
    h1 { color: #1a1a2e; border-bottom: 2px solid #e94560; padding-bottom: 10px; }
    .info { background: #f5f5f5; padding: 20px; border-radius: 8px; margin-top: 20px; }
  </style>
</head>
<body>
  <h1>Hello from Docbiner!</h1>
  <p>This is a sample document generated with the Docbiner playground.</p>
  <div class="info">
    <p><strong>Format:</strong> PDF</p>
    <p><strong>Generated at:</strong> <span id="date"></span></p>
  </div>
  <script>
    document.getElementById('date').textContent = new Date().toLocaleString();
  </script>
</body>
</html>`;

interface PlaygroundEditorProps {
  value: string;
  onChange: (value: string) => void;
}

export { DEFAULT_HTML };

export function PlaygroundEditor({ value, onChange }: PlaygroundEditorProps) {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);

  const handleMount: OnMount = useCallback((editor) => {
    editorRef.current = editor;
  }, []);

  const handleChange = useCallback(
    (val: string | undefined) => {
      onChange(val ?? '');
    },
    [onChange],
  );

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center border-b border-border px-4 py-2">
        <p className="text-xs font-medium text-muted-foreground uppercase">
          HTML Editor
        </p>
      </div>
      <div className="flex-1">
        <Editor
          defaultLanguage="html"
          value={value}
          onChange={handleChange}
          onMount={handleMount}
          theme="vs-dark"
          loading={
            <div className="flex h-full items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          }
          options={{
            minimap: { enabled: false },
            fontSize: 13,
            lineNumbers: 'on',
            wordWrap: 'on',
            scrollBeyondLastLine: false,
            automaticLayout: true,
            padding: { top: 12 },
            tabSize: 2,
            formatOnPaste: true,
          }}
        />
      </div>
    </div>
  );
}
