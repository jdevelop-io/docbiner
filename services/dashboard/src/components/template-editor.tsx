'use client';

import { useState, useCallback } from 'react';
import Editor from '@monaco-editor/react';
import { Code, Paintbrush, Database } from 'lucide-react';
import { cn } from '@/lib/utils';

type Tab = 'html' | 'css' | 'data';

interface TemplateEditorProps {
  htmlContent: string;
  cssContent: string;
  sampleData: string;
  onHtmlChange: (value: string) => void;
  onCssChange: (value: string) => void;
  onSampleDataChange: (value: string) => void;
}

const tabs: { id: Tab; label: string; icon: typeof Code }[] = [
  { id: 'html', label: 'HTML', icon: Code },
  { id: 'css', label: 'CSS', icon: Paintbrush },
  { id: 'data', label: 'Sample Data', icon: Database },
];

export function TemplateEditor({
  htmlContent,
  cssContent,
  sampleData,
  onHtmlChange,
  onCssChange,
  onSampleDataChange,
}: TemplateEditorProps) {
  const [activeTab, setActiveTab] = useState<Tab>('html');

  const handleEditorChange = useCallback(
    (value: string | undefined) => {
      const val = value ?? '';
      switch (activeTab) {
        case 'html':
          onHtmlChange(val);
          break;
        case 'css':
          onCssChange(val);
          break;
        case 'data':
          onSampleDataChange(val);
          break;
      }
    },
    [activeTab, onHtmlChange, onCssChange, onSampleDataChange],
  );

  const editorValue = activeTab === 'html' ? htmlContent : activeTab === 'css' ? cssContent : sampleData;
  const editorLanguage = activeTab === 'data' ? 'json' : activeTab;

  return (
    <div className="flex h-full flex-col overflow-hidden rounded-lg border border-border bg-card">
      {/* Tabs */}
      <div className="flex border-b border-border bg-muted/50">
        {tabs.map((tab) => {
          const Icon = tab.icon;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'flex items-center gap-1.5 px-4 py-2 text-sm font-medium transition-colors',
                activeTab === tab.id
                  ? 'border-b-2 border-primary bg-card text-foreground'
                  : 'text-muted-foreground hover:text-foreground',
              )}
            >
              <Icon className="h-3.5 w-3.5" />
              {tab.label}
            </button>
          );
        })}
      </div>

      {/* Editor */}
      <div className="flex-1 min-h-0">
        <Editor
          height="100%"
          language={editorLanguage}
          value={editorValue}
          onChange={handleEditorChange}
          theme="vs-dark"
          options={{
            minimap: { enabled: false },
            fontSize: 14,
            lineNumbers: 'on',
            wordWrap: 'on',
            automaticLayout: true,
            scrollBeyondLastLine: false,
            tabSize: 2,
            padding: { top: 12 },
          }}
          loading={
            <div className="flex h-full items-center justify-center">
              <p className="text-sm text-muted-foreground">Loading editor...</p>
            </div>
          }
        />
      </div>
    </div>
  );
}
