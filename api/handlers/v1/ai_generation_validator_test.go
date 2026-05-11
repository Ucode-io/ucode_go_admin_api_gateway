package v1

import (
	"testing"

	"ucode/ucode_go_api_gateway/api/models"
)

// ============================================================================
// VALIDATOR UNIT TESTS
//
// These tests simulate real error classes from past generation failures.
// Run with: go test -v ./api/handlers/v1/ -run TestValidat
// ============================================================================

// TestValidate_MissingFile — import from a file that doesn't exist in output.
// Real failure: feature chunk imports from '@/components/ui/calendar' but calendar.tsx was never generated.
func TestValidate_MissingFile(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path:    "src/pages/OrdersPage.tsx",
			Content: `import React from 'react';\nimport { Calendar } from '@/components/ui/calendar';\nexport default function OrdersPage() { return <Calendar />; }`,
		},
		{
			Path:    "src/components/ui/button.tsx",
			Content: `import React from 'react';\nexport const Button = React.forwardRef(() => null);\nButton.displayName = 'Button';`,
		},
	}

	errors := validateGeneratedProject(files, nil)
	found := false
	for _, e := range errors {
		if e.Severity == "error" && contains(e.Message, "does not exist") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for missing '@/components/ui/calendar', got %d errors: %v", len(errors), errors)
	}
}

// TestValidate_MissingExport — import a named export that doesn't exist.
// Real failure: QuoteStatusBadge was imported from Badge.tsx but Badge.tsx only exports Badge + BadgeProps.
func TestValidate_MissingExport(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path: "src/components/ui/badge.tsx",
			Content: `import React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';

export const badgeVariants = cva('inline-flex items-center');

export interface BadgeProps extends React.HTMLAttributes<HTMLDivElement>,
  VariantProps<typeof badgeVariants> {}

export const Badge = React.forwardRef<HTMLDivElement, BadgeProps>(
  ({ className, variant, ...props }, ref) => (
    <div ref={ref} className={badgeVariants({ variant })} {...props} />
  )
);
Badge.displayName = 'Badge';`,
		},
		{
			Path: "src/features/quotes/components/QuoteList.tsx",
			Content: `import React from 'react';
import { Badge, QuoteStatusBadge } from '@/components/ui/badge';
export function QuoteList() { return <Badge />; }`,
		},
	}

	errors := validateGeneratedProject(files, nil)
	found := false
	for _, e := range errors {
		if e.Severity == "error" && contains(e.Message, "QuoteStatusBadge") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for missing export 'QuoteStatusBadge', got %d errors: %v", len(errors), errors)
	}
}

// TestValidate_ValidProject — no errors for a correctly wired project.
func TestValidate_ValidProject(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path: "src/components/ui/button.tsx",
			Content: `import React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';

export const buttonVariants = cva('btn');

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement>,
  VariantProps<typeof buttonVariants> {}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, ...props }, ref) => (
    <button ref={ref} {...props} />
  )
);
Button.displayName = 'Button';`,
		},
		{
			Path: "src/components/ui/badge.tsx",
			Content: `import React from 'react';
export function Badge({ children }: { children: React.ReactNode }) { return <span>{children}</span>; }`,
		},
		{
			Path: "src/pages/DashboardPage.tsx",
			Content: `import React from 'react';
import { Button, buttonVariants } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
export default function DashboardPage() { return <div><Button /><Badge>OK</Badge></div>; }`,
		},
		{
			Path: "src/App.tsx",
			Content: `import React from 'react';
import './index.css';
import DashboardPage from './pages/DashboardPage';
const baseUrl = import.meta.env.VITE_API_BASE_URL;
export default function App() { return <DashboardPage />; }`,
		},
		{
			Path: "src/index.css",
			Content: `:root { --primary: 220 80% 50%; }`,
		},
		{
			Path: ".env",
			Content: "VITE_API_BASE_URL=https://api.example.com\nVITE_X_API_KEY=test-key",
		},
	}

	errors := validateGeneratedProject(files, nil)
	errorCount := 0
	for _, e := range errors {
		if e.Severity == "error" {
			errorCount++
			t.Errorf("unexpected error: [%s] %s", e.File, e.Message)
		}
	}
	if errorCount > 0 {
		t.Errorf("expected 0 errors for valid project, got %d", errorCount)
	}
}

// TestValidate_EnvMismatch — env var used in code but not defined anywhere.
func TestValidate_EnvMismatch(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path:    "src/config/axios.ts",
			Content: `import axios from 'axios';\nconst api = axios.create({ baseURL: import.meta.env.VITE_CUSTOM_URL });\nexport default api;`,
		},
		{
			Path:    ".env",
			Content: "VITE_API_BASE_URL=https://api.example.com",
		},
	}

	errors := validateGeneratedProject(files, nil)
	found := false
	for _, e := range errors {
		if contains(e.Message, "VITE_CUSTOM_URL") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning for undefined VITE_CUSTOM_URL, got %d errors: %v", len(errors), errors)
	}
}

// TestValidate_RelativeImport — relative import './utils' resolves correctly.
func TestValidate_RelativeImport(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path:    "src/features/orders/api.ts",
			Content: `import { OrderType } from './types';\nexport function getOrders() { return null; }`,
		},
		{
			Path:    "src/features/orders/types.ts",
			Content: `export interface OrderType { id: string; }`,
		},
	}

	errors := validateGeneratedProject(files, nil)
	for _, e := range errors {
		if e.Severity == "error" {
			t.Errorf("unexpected error for valid relative import: [%s] %s", e.File, e.Message)
		}
	}
}

// TestValidate_RelativeImportMissing — relative import to non-existent file.
func TestValidate_RelativeImportMissing(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path:    "src/features/orders/api.ts",
			Content: `import { OrderType } from './types';\nimport { formatOrder } from './formatters';\nexport function getOrders() { return null; }`,
		},
		{
			Path:    "src/features/orders/types.ts",
			Content: `export interface OrderType { id: string; }`,
		},
	}

	errors := validateGeneratedProject(files, nil)
	found := false
	for _, e := range errors {
		if e.Severity == "error" && contains(e.Message, "formatters") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error for missing './formatters', got %d errors: %v", len(errors), errors)
	}
}

// TestValidate_TemplateFilesSkipped — imports from template files should NOT be flagged.
func TestValidate_TemplateFilesSkipped(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path: "src/features/users/api.ts",
			Content: `import { useApiQuery, useApiMutation } from '@/hooks/useApi';
import { extractList, extractCount } from '@/lib/apiUtils';
import { cn } from '@/lib/utils';
export function useUsers() { return useApiQuery(['users'], '/v2/items/users'); }`,
		},
	}

	errors := validateGeneratedProject(files, nil)
	for _, e := range errors {
		if e.Severity == "error" {
			t.Errorf("template file import flagged as error: [%s] %s", e.File, e.Message)
		}
	}
}

// TestValidate_ExportBraces — export { X, Y, Z } pattern detected.
func TestValidate_ExportBraces(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path: "src/lib/helpers.ts",
			Content: `function formatDate(d: string) { return d; }
function formatCurrency(n: number) { return n.toString(); }
export { formatDate, formatCurrency };`,
		},
		{
			Path:    "src/pages/ReportPage.tsx",
			Content: `import { formatDate, formatCurrency } from '@/lib/helpers';\nexport default function ReportPage() { return null; }`,
		},
	}

	errors := validateGeneratedProject(files, nil)
	for _, e := range errors {
		if e.Severity == "error" {
			t.Errorf("unexpected error for export {} pattern: [%s] %s", e.File, e.Message)
		}
	}
}

// TestValidate_DisplayNamePattern — React.forwardRef with displayName should be detected as export.
func TestValidate_DisplayNamePattern(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path: "src/components/ui/input.tsx",
			Content: `import React from 'react';
export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {}
export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, ...props }, ref) => <input ref={ref} {...props} />
);
Input.displayName = 'Input';`,
		},
		{
			Path:    "src/features/users/UserForm.tsx",
			Content: `import { Input, InputProps } from '@/components/ui/input';\nexport function UserForm() { return <Input />; }`,
		},
	}

	errors := validateGeneratedProject(files, nil)
	for _, e := range errors {
		if e.Severity == "error" {
			t.Errorf("unexpected error for forwardRef/displayName: [%s] %s", e.File, e.Message)
		}
	}
}

// TestBuildUIKitAPISummary — verifies the API summary extraction.
func TestBuildUIKitAPISummary(t *testing.T) {
	files := []models.ProjectFile{
		{
			Path: "src/components/ui/button.tsx",
			Content: `import React from 'react';
export const buttonVariants = cva('btn');
export interface ButtonProps {}
export const Button = React.forwardRef(() => null);
Button.displayName = 'Button';`,
		},
	}

	summary := buildUIKitAPISummary(files)
	if summary == "" {
		t.Error("expected non-empty UI Kit API summary")
	}
	if !contains(summary, "buttonVariants") {
		t.Error("expected summary to contain 'buttonVariants'")
	}
	if !contains(summary, "ButtonProps") {
		t.Error("expected summary to contain 'ButtonProps'")
	}
	if !contains(summary, "Button") {
		t.Error("expected summary to contain 'Button'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
