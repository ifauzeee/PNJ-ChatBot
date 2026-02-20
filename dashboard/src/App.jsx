import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Activity,
  Users,
  Search,
  Clock,
  Database,
  Cpu,
  ShieldCheck,
  Zap,
  RotateCcw,
  LayoutDashboard
} from 'lucide-react';

const getApiUrl = () => {
  // 1. Priority: Vite environment variable (set during build)
  if (import.meta.env.VITE_API_BASE_URL) {
    return import.meta.env.VITE_API_BASE_URL;
  }

  // 2. Dynamic fallback: current browser hostname on port 8080
  if (typeof window !== 'undefined') {
    return `${window.location.protocol}//${window.location.hostname}:8080`;
  }

  return 'http://localhost:8080';
};

const API_BASE_URL = getApiUrl();

function App() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [lastUpdated, setLastUpdated] = useState(new Date());

  const fetchData = async () => {
    try {
      const response = await axios.get(`${API_BASE_URL}/health`);
      setData(response.data);
      setError(null);
    } catch (err) {
      console.error('Error fetching dashboard data:', err);
      setError('Cannot connect to Bot API');
    } finally {
      setLoading(false);
      setLastUpdated(new Date());
    }
  };

  useEffect(() => {
    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, []);

  const formatBytes = (bytes) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getUptime = (seconds) => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    return `${h}h ${m}m`;
  };

  return (
    <div className="dashboard-container">
      <header className="header">
        <div className="title-group">
          <h1>PNJ Bot Dashboard</h1>
          <p>Real-time system monitoring & diagnostics</p>
        </div>
        <div style={{ textAlign: 'right' }}>
          <div className="status-item" style={{ background: 'var(--glass-bg)', padding: '0.5rem 1rem' }}>
            <div className={`status-indicator ${error ? 'off' : 'on'}`}></div>
            <span>{error ? 'API Disconnected' : 'Live Connection'}</span>
          </div>
          <p className="stat-subtext" style={{ marginTop: '0.5rem' }}>
            Last updated: {lastUpdated.toLocaleTimeString()}
          </p>
        </div>
      </header>

      <main>
        <div className="stats-grid">
          <StatCard
            label="Active Connections"
            value={data?.stats?.total_user_online || 0}
            icon={<Users size={20} />}
            subtext="Real-time users connected"
            delay="0.1s"
          />
          <StatCard
            label="In Queue"
            value={data?.stats?.total_user_queue || 0}
            icon={<Search size={20} />}
            subtext="Searching for partners"
            delay="0.2s"
          />
          <StatCard
            label="Heap Memory"
            value={formatBytes(data?.system?.heap_alloc || 0)}
            icon={<Cpu size={20} />}
            subtext={`Stack: ${formatBytes(data?.system?.stack_in_use || 0)}`}
            delay="0.3s"
          />
          <StatCard
            label="Uptime"
            value={getUptime(data?.system?.uptime_seconds || 0)}
            icon={<Clock size={20} />}
            subtext={`Goroutines: ${data?.system?.goroutines || 0}`}
            delay="0.4s"
          />
        </div>

        <section className="status-section">
          <h2 className="section-title">
            <Activity size={24} className="text-primary" />
            Infrastructure Status
          </h2>
          <div className="status-grid">
            <StatusItem
              label="Bot Core"
              status={data?.status || 'offline'}
              desc="Telegram API processing & handlers"
            />
            <StatusItem
              label="Database"
              status={data?.database || 'offline'}
              desc="Postgres/SQLite persistence layer"
            />
            <StatusItem
              label="Redis Cache"
              status={data?.redis || 'offline'}
              desc="Queue management and user state"
            />
          </div>
        </section>

        <section className="status-section">
          <h2 className="section-title">
            <ShieldCheck size={24} style={{ color: 'var(--success)' }} />
            Security & Performance
          </h2>
          <div className="status-grid">
            <FeatureBox
              icon={<Zap size={20} style={{ color: 'var(--warning)' }} />}
              label="Rate Limiting"
              value="ACTIVE"
            />
            <FeatureBox
              icon={<ShieldCheck size={20} style={{ color: 'var(--primary)' }} />}
              label="Sentry Tracking"
              value="ENABLED"
            />
            <FeatureBox
              icon={<RotateCcw size={20} style={{ color: 'var(--secondary)' }} />}
              label="Auto Recovery"
              value="ON"
            />
          </div>
        </section>
      </main>

      {error && (
        <div style={{
          position: 'fixed',
          bottom: '2rem',
          right: '2rem',
          background: 'var(--error)',
          padding: '1rem 2rem',
          borderRadius: '1rem',
          boxShadow: '0 10px 30px rgba(0,0,0,0.5)',
          animation: 'fadeInUp 0.3s ease-out'
        }}>
          ⚠️ {error}
        </div>
      )}
    </div>
  );
}

function StatCard({ label, value, icon, subtext, delay }) {
  return (
    <div className="stat-card" style={{ animationDelay: delay }}>
      <div className="stat-label">
        <span style={{ color: 'var(--primary)' }}>{icon}</span>
        {label}
      </div>
      <div className="stat-value">{value}</div>
      <div className="stat-subtext">{subtext}</div>
    </div>
  );
}

function StatusItem({ label, status, desc }) {
  const isOn = status === 'ok' || status === 'up' || status === 'connected';
  return (
    <div className="status-item">
      <div className={`status-indicator ${isOn ? 'on' : 'off'}`}></div>
      <div className="status-info">
        <h4>{label}</h4>
        <p>{desc}</p>
      </div>
    </div>
  );
}

function FeatureBox({ icon, label, value }) {
  return (
    <div className="status-item" style={{ justifyContent: 'space-between' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
        {icon}
        <h4>{label}</h4>
      </div>
      <span style={{ fontSize: '0.75rem', fontWeight: 800, color: 'var(--primary)', border: '1px solid var(--primary)', padding: '0.25rem 0.5rem', borderRadius: '0.5rem' }}>
        {value}
      </span>
    </div>
  );
}

export default App;
