import "./index.css";

const navItems = ["Dashboard", "Endpoint", "Providers", "Usage"];

const statusItems = [
  { label: "Gateway status", value: "Awaiting server data", tone: "bg-sky-500" },
  { label: "Provider health", value: "No checks loaded", tone: "bg-emerald-500" },
  { label: "Request flow", value: "Idle", tone: "bg-amber-500" }
];

function App() {
  return (
    <div className="min-h-screen bg-zinc-50 text-zinc-950">
      <aside className="fixed inset-y-0 left-0 hidden w-64 border-r border-zinc-200 bg-white px-5 py-6 lg:block">
        <h1 className="text-xl font-semibold tracking-normal">g0router</h1>
        <p className="mt-1 text-sm text-zinc-500">Control plane</p>

        <nav aria-label="Primary" className="mt-8 space-y-1">
          {navItems.map((item) => (
            <a
              key={item}
              href="#"
              aria-current={item === "Dashboard" ? "page" : undefined}
              className="block rounded-md px-3 py-2 text-sm font-medium text-zinc-600 transition hover:bg-zinc-100 hover:text-zinc-950 aria-[current=page]:bg-zinc-950 aria-[current=page]:text-white"
            >
              {item}
            </a>
          ))}
        </nav>
      </aside>

      <main className="lg:pl-64">
        <header className="border-b border-zinc-200 bg-white px-5 py-4 lg:px-8">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <p className="text-sm font-medium text-zinc-500">Dashboard</p>
              <h2 className="text-2xl font-semibold tracking-normal">Gateway overview</h2>
            </div>
            <div className="inline-flex w-fit items-center gap-2 rounded-md border border-zinc-200 px-3 py-2 text-sm font-medium text-zinc-700">
              <span className="h-2 w-2 rounded-full bg-emerald-500" />
              Local control plane
            </div>
          </div>
        </header>

        <section className="px-5 py-6 lg:px-8">
          <div className="grid gap-4 md:grid-cols-3">
            {statusItems.map((item) => (
              <article key={item.label} className="rounded-md border border-zinc-200 bg-white p-5">
                <div className="flex items-center justify-between gap-3">
                  <h3 className="text-sm font-semibold text-zinc-700">{item.label}</h3>
                  <span className={`h-2.5 w-2.5 rounded-full ${item.tone}`} />
                </div>
                <p className="mt-4 text-xl font-semibold tracking-normal">{item.value}</p>
                <p className="mt-2 text-sm leading-6 text-zinc-500">Ready for Phase 10 data wiring.</p>
              </article>
            ))}
          </div>

          <div className="mt-6 rounded-md border border-zinc-200 bg-white">
            <div className="border-b border-zinc-200 px-5 py-4">
              <h3 className="text-base font-semibold">Operational queue</h3>
            </div>
            <div className="grid gap-px bg-zinc-200 md:grid-cols-3">
              {["Connections", "Models", "Recent requests"].map((label) => (
                <div key={label} className="bg-white p-5">
                  <p className="text-sm font-medium text-zinc-500">{label}</p>
                  <p className="mt-3 text-3xl font-semibold tracking-normal">0</p>
                </div>
              ))}
            </div>
          </div>
        </section>
      </main>
    </div>
  );
}

export default App;
