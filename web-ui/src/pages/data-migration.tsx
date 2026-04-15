import { useState, useRef } from "react";
import { post } from "@/lib/api";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Separator } from "@/components/ui/separator";
import { Download, Upload, Play, Loader2, Trash2 } from "lucide-react";
import type { ImportResult, SqlResult } from "@/types";

export default function DataMigrationPage() {
  const [importing, setImporting] = useState(false);
  const [importResult, setImportResult] = useState<ImportResult | null>(null);
  const [sql, setSql] = useState("");
  const [sqlRunning, setSqlRunning] = useState(false);
  const [sqlResult, setSqlResult] = useState<SqlResult | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  async function handleExport() {
    try {
      const res = await fetch("/api/export", { credentials: "include" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `ruleflow-backup-${new Date().toISOString().slice(0, 10)}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      toast.success("Export downloaded");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Export failed");
    }
  }

  async function handleImport() {
    const file = fileRef.current?.files?.[0];
    if (!file) return;
    setImporting(true);
    setImportResult(null);
    try {
      const formData = new FormData();
      formData.append("file", file);
      const res = await fetch("/api/import", { method: "POST", body: formData, credentials: "include" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json = await res.json();
      if (json.code !== 0) throw new Error(json.msg);
      setImportResult(json.data);
      toast.success("Import complete");
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "Import failed");
    } finally {
      setImporting(false);
    }
  }

  async function handleSql() {
    if (!sql.trim()) return;
    setSqlRunning(true);
    setSqlResult(null);
    try {
      const data = await post<SqlResult>("/api/admin/exec-sql", { sql: sql.trim() });
      setSqlResult(data);
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : "SQL failed");
    } finally {
      setSqlRunning(false);
    }
  }

  const importCategories = importResult
    ? [
        { label: "Subscriptions", ...importResult.subscriptions },
        { label: "Nodes", ...importResult.manual_nodes },
        { label: "Templates", ...importResult.templates },
        { label: "Policies", ...importResult.config_policies },
        { label: "Rule Sources", ...importResult.rule_sources },
      ]
    : [];

  return (
    <div className="space-y-6 p-6">
      <div>
        <h1 className="font-heading text-2xl font-bold tracking-tight">Data Migration</h1>
        <p className="text-sm text-muted-foreground">Import and export configuration data</p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Export */}
        <Card>
          <CardHeader><CardTitle className="text-base">Export Data</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">Download all data as a JSON backup file.</p>
            <Button onClick={handleExport}><Download className="size-4 mr-1.5" /> Export</Button>
          </CardContent>
        </Card>

        {/* Import */}
        <Card>
          <CardHeader><CardTitle className="text-base">Import Data</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <p className="text-sm text-muted-foreground">Upload a JSON backup file to restore data.</p>
            <div className="flex gap-2">
              <input ref={fileRef} type="file" accept=".json" className="text-sm file:mr-2 file:rounded-md file:border-0 file:bg-primary file:px-3 file:py-1.5 file:text-xs file:text-primary-foreground file:cursor-pointer" />
              <Button onClick={handleImport} disabled={importing}>
                {importing ? <Loader2 className="size-4 mr-1.5 animate-spin" /> : <Upload className="size-4 mr-1.5" />} Import
              </Button>
            </div>
            {importResult && (
              <div className="space-y-2">
                <Separator />
                <div className="grid grid-cols-5 gap-2 text-center text-xs">
                  <span className="font-medium">Category</span>
                  <span className="text-emerald-500">New</span>
                  <span className="text-sky-500">Updated</span>
                  <span className="text-muted-foreground">Skipped</span>
                  <span className="text-destructive">Errors</span>
                </div>
                {importCategories.map((c) => (
                  <div key={c.label} className="grid grid-cols-5 gap-2 text-center text-xs">
                    <span className="font-medium text-left">{c.label}</span>
                    <Badge variant="outline" className="text-emerald-500 justify-center">{c.created}</Badge>
                    <Badge variant="outline" className="text-sky-500 justify-center">{c.updated}</Badge>
                    <Badge variant="outline" className="justify-center">{c.skipped}</Badge>
                    <Badge variant="outline" className="text-destructive justify-center">{c.errors?.length || 0}</Badge>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* SQL Executor */}
      <Card>
        <CardHeader><CardTitle className="text-base">SQL Executor</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2">
            <Label>SQL Query</Label>
            <Textarea
              value={sql}
              onChange={(e) => setSql(e.target.value)}
              rows={4}
              className="font-mono text-xs"
              placeholder="SELECT * FROM ..."
              onKeyDown={(e) => { if ((e.metaKey || e.ctrlKey) && e.key === "Enter") handleSql(); }}
            />
          </div>
          <div className="flex gap-2">
            <Button onClick={handleSql} disabled={sqlRunning || !sql.trim()}>
              {sqlRunning ? <Loader2 className="size-4 mr-1.5 animate-spin" /> : <Play className="size-4 mr-1.5" />} Run
            </Button>
            <Button variant="outline" onClick={() => { setSql(""); setSqlResult(null); }}><Trash2 className="size-4 mr-1.5" /> Clear</Button>
          </div>
          {sqlResult && (
            <div className="space-y-2">
              <Separator />
              {sqlResult.type === "select" && sqlResult.rows?.length > 0 ? (
                <div className="max-h-96 overflow-auto rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>{sqlResult.columns.map((c) => <TableHead key={c} className="text-xs">{c}</TableHead>)}</TableRow>
                    </TableHeader>
                    <TableBody>
                      {sqlResult.rows.slice(0, 500).map((row, i) => (
                        <TableRow key={i}>
                          {sqlResult.columns.map((c) => (
                            <TableCell key={c} className="text-xs font-mono max-w-[200px] truncate">{String(row[c] ?? "NULL")}</TableCell>
                          ))}
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  {sqlResult.type === "select" ? "No rows returned" : `${sqlResult.rows_affected} row(s) affected`}
                </p>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
