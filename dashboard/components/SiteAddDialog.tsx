"use client"

import { useState } from "react"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "./ui/dialog";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { fetchWithAuth } from "@/lib/api";

type Site = {
  id: string;
  name: string;
  created_at: string;
  api_key?: string;
};

type AddSiteDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSiteCreated?: (site: Site) => void;
};

export default function AddSiteDialog({ open, onOpenChange, onSiteCreated }: AddSiteDialogProps) {
  const [isSubmitted, setSubmitted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [isCopied, setIsCopied] = useState(false);
  const [trackingId, setTrackingId] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [error, setError] = useState<string | null>(null);

  const [siteName, setSiteName] = useState("");
  const [siteUrl, setSiteUrl] = useState("");

  const codeSnippet = `<script src="https://cdn.your-engine.com/tracker.js" data-site-id="${trackingId}" data-api-key="${apiKey}" defer></script>`;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      const response = await fetchWithAuth("/v1/sites", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: siteName.trim(), url: siteUrl.trim() }),
      });

      if (!response.ok) {
        const payload = (await response.json().catch(() => null)) as
          | { error?: string }
          | { error?: { message?: string } }
          | null;
        const message =
          typeof payload?.error === "string"
            ? payload.error
            : payload?.error?.message ?? "Failed to create site";
        throw new Error(message);
      }

      const site = (await response.json()) as Site;
      setTrackingId(site.id);
      setApiKey(site.api_key || "");
      setSubmitted(true);
      onSiteCreated?.(site);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create site");
    } finally {
      setLoading(false);
    }
  };

  const copyToClipboard = async () => {
    await navigator.clipboard.writeText(codeSnippet);
    setIsCopied(true);
    setTimeout(() => setIsCopied(false), 2000);
  }

  const handleClose = (isOpen: boolean) => {
    onOpenChange(isOpen);
    if (!isOpen) {
      setTimeout(() => {
        setSubmitted(false);
        setSiteName("");
        setSiteUrl("");
        setTrackingId("");
        setApiKey("");
        setError(null);
        setIsCopied(false);
      }, 300);
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[450px] rounded-md">
        {!isSubmitted ? (
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle className="font-serif text-2xl tracking-wide">Add a New Site</DialogTitle>
              <DialogDescription className="text-sm">
                Enter your website details to generate a tracking snippet.
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-6">
              <div className="grid gap-2">
                <Label htmlFor="name" className="text-sm">Site Name</Label>
                <Input
                  id="name"
                  placeholder="e.g. My Awesome Blog"
                  value={siteName}
                  onChange={(e) => setSiteName(e.target.value)}
                  required
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="url" className="text-sm">Domain URL</Label>
                <Input
                  id="url"
                  placeholder="e.g. https://myblog.com"
                  value={siteUrl}
                  onChange={(e) => setSiteUrl(e.target.value)}
                  required
                  className="text-sm"
                />
              </div>
              {error ? <p className="text-sm text-red-600">{error}</p> : null}
            </div>

            <DialogFooter>
              <Button size='lg' type="button" variant="outline" className="text-sm px-6 cursor-pointer" onClick={() => handleClose(false)}>
                Cancel
              </Button>
              <Button size='lg' type="submit" className="text-sm px-6 cursor-pointer" disabled={loading || !siteName || !siteUrl}>
                {loading ? "Creating..." : "Create Site"}
              </Button>
            </DialogFooter>
          </form>
        ) : (
          <>
            <DialogHeader>
              <div className="flex items-center gap-2 text-green-600 mb-1">
                {/*<Check size={20} />*/}
                <span className="font-semibold text-sm tracking-wide uppercase">Site Created</span>
              </div>
              <DialogTitle className="text-2xl font-serif">Install your tracker</DialogTitle>
              <DialogDescription className="text-sm">
                Paste this snippet into the <code>&lt;head&gt;</code> of your website.
              </DialogDescription>
            </DialogHeader>

            <div className="py-4">
              {/* Code Snippet Container */}
              <div className="relative group rounded-md bg-zinc-950 p-4 font-mono text-sm text-zinc-50 border border-zinc-800">
                <div className="flex items-center justify-between mb-2 text-zinc-400 border-b border-zinc-800 pb-2">
                  <div className="flex items-center gap-2 text-xs">
                    {/*<TerminalSquare size={14} />*/}
                    <span>HTML</span>
                  </div>
                  {/* Copy Button */}
                  <button
                    onClick={copyToClipboard}
                    className="flex items-center gap-1.5 hover:text-white transition-colors p-1"
                    title="Copy to clipboard"
                  >
                    {/*{isCopied ? <Check size={14} className="text-green-400" /> : <Copy size={14} />}*/}
                    <span className="text-xs">{isCopied ? "Copied!" : "Copy"}</span>
                  </button>
                </div>
                
                <pre className="overflow-x-auto whitespace-pre-wrap word-break-all text-xs leading-relaxed text-zinc-300">
                  <code>{codeSnippet}</code>
                </pre>
              </div>
            </div>

            <DialogFooter>
              <Button size='lg' onClick={() => handleClose(false)} className="w-full sm:w-auto cursor-pointer text-sm px-4">
                Done
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
