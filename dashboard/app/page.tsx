import Link from "next/link";

export default function LandingPage() {
  return (
    <main className="min-h-screen relative selection:bg-foreground selection:text-background flex flex-col">
      {/* Optional: Subtle SVG Noise Texture Overlay 
        This gives the dark background a tactile, premium feel 
      */}
      <div 
        className="pointer-events-none fixed inset-0 z-50 opacity-[0.03]"
        style={{ backgroundImage: `url("data:image/svg+xml,%3Csvg viewBox='0 0 200 200' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noiseFilter'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.65' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noiseFilter)'/%3E%3C/svg%3E")` }}
      />

      {/* Navigation */}
      <nav className="w-full flex items-center justify-between p-6 md:p-12 z-10">
        <div className="font-sans font-medium tracking-tight text-sm flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-foreground animate-pulse" />
          Analytics Engine
        </div>
        <div className="flex gap-6 items-center text-sm">
          <Link href="https://github.com/AgnivaSengupta/go-analytics-ingestion-engine" target="_blank" className="text-muted hover:text-foreground transition-colors">
            Architecture
          </Link>
          <Link href="/login" className="border border-border px-4 py-2 hover:bg-foreground hover:text-background transition-colors duration-300">
            Developer Login
          </Link>
        </div>
      </nav>

      {/* Hero Section */}
      <section className="flex-1 flex flex-col justify-center px-6 md:px-12 z-10 mt-12 md:mt-0">
        <div className="max-w-4xl">
          <h2 className="text-muted font-sans text-sm tracking-widest uppercase mb-6 border-l border-border pl-4">
            High-Throughput Ingestion Pipeline
          </h2>
          
          <h1 className="font-serif text-6xl md:text-8xl leading-[0.9] tracking-tight mb-8">
            Measure <br className="hidden md:block" />
            Everything. <br />
            <span className="text-muted">Wait for nothing.</span>
          </h1>
          
          <p className="font-sans text-muted max-w-xl text-lg md:text-xl leading-relaxed mb-12">
            A distributed analytics ingestion engine engineered in Go. Utilizing a Redis buffer queue and PostgreSQL background workers to process thousands of events per second with sub-10ms API latency.
          </p>

          <div className="flex flex-col sm:flex-row gap-4">
            <Link 
              href="/sites/demo/overview" 
              className="bg-foreground text-background px-8 py-4 font-medium text-sm flex items-center justify-center gap-2 hover:opacity-90 transition-opacity"
            >
              View Demo Dashboard
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="square" strokeLinejoin="miter">
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </Link>
            <Link 
              href="https://github.com/AgnivaSengupta/go-analytics-ingestion-engine" 
              target="_blank"
              className="border border-border px-8 py-4 font-medium text-sm text-center hover:bg-border/50 transition-colors"
            >
              View GitHub Repo
            </Link>
          </div>
        </div>
      </section>

      {/* Metrics / Features Grid */}
      <section className="grid grid-cols-1 md:grid-cols-3 border-t border-border z-10 mt-24">
        <div className="p-6 md:p-12 border-b md:border-b-0 md:border-r border-border">
          <div className="font-sans text-muted text-xs uppercase tracking-widest mb-4">Ingestion Target</div>
          <div className="font-serif text-5xl md:text-6xl mb-2">5,000<span className="text-2xl text-muted font-sans"> /sec</span></div>
          <p className="text-sm text-muted">Sustained event processing capability powered by non-blocking Go Fiber routines.</p>
        </div>
        
        <div className="p-6 md:p-12 border-b md:border-b-0 md:border-r border-border">
          <div className="font-sans text-muted text-xs uppercase tracking-widest mb-4">API Latency</div>
          <div className="font-serif text-5xl md:text-6xl mb-2">&lt; 15<span className="text-2xl text-muted font-sans"> ms</span></div>
          <p className="text-sm text-muted">p95 response time achieved by decoupling HTTP ingestion from database persistence.</p>
        </div>

        <div className="p-6 md:p-12 bg-border/10">
          <div className="font-sans text-muted text-xs uppercase tracking-widest mb-4">Architecture</div>
          <ul className="text-sm space-y-3 font-mono text-muted">
            <li className="flex justify-between border-b border-border/50 pb-2"><span>API Server</span> <span className="text-foreground">Go / Fiber</span></li>
            <li className="flex justify-between border-b border-border/50 pb-2"><span>Buffer</span> <span className="text-foreground">Redis</span></li>
            <li className="flex justify-between border-b border-border/50 pb-2"><span>Persistence</span> <span className="text-foreground">PostgreSQL</span></li>
            <li className="flex justify-between"><span>Dashboard</span> <span className="text-foreground">Next.js</span></li>
          </ul>
        </div>
      </section>
    </main>
  );
}