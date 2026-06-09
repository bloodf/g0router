export default function App() {
  return (
    <main
      style={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '2rem',
        textAlign: 'center',
        fontFamily: 'system-ui, -apple-system, "Segoe UI", Roboto, sans-serif',
        colorScheme: 'light dark',
      }}
    >
      <h1 style={{ fontSize: '2rem', fontWeight: 600, margin: 0 }}>
        g0router v2.0
      </h1>
      <p style={{ marginTop: '0.5rem', color: '#6b7280' }}>coming soon</p>
      <a
        href="/api/health"
        target="_blank"
        rel="noreferrer"
        style={{
          marginTop: '1.5rem',
          color: '#2563eb',
          textDecoration: 'underline',
        }}
      >
        /api/health
      </a>
    </main>
  )
}
