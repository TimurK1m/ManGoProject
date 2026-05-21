import React, { useEffect, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { api } from '../api';
import { ArrowLeft, Activity, Clock, ServerCrash, CheckCircle2 } from 'lucide-react';

interface Stats {
  uptime_percent: number;
  avg_response_time: number;
  total_checks: number;
}

interface Check {
  id: number;
  status: string;
  response_time: number;
  created_at: string;
}

const ServiceDetails: React.FC = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const [stats, setStats] = useState<Stats | null>(null);
  const [checks, setChecks] = useState<Check[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [statsRes, checksRes] = await Promise.all([
          api.get(`/services/${id}/stats`),
          api.get(`/services/${id}/checks`)
        ]);
        setStats(statsRes.data);
        setChecks(checksRes.data.checks || []);
      } catch (err) {
        console.error(err);
        navigate('/'); // Go back if error (e.g. 403 or 404)
      } finally {
        setLoading(false);
      }
    };
    
    fetchData();
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, [id, navigate]);

  if (loading) return <div className="text-center py-20">Loading...</div>;

  return (
    <div>
      <div className="mb-6">
        <Link to="/" className="text-slate-400 hover:text-white inline-flex items-center gap-2 mb-4 transition-colors">
          <ArrowLeft size={16} /> Back to Dashboard
        </Link>
        <h1 className="text-3xl font-bold">Service Overview</h1>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="glass-panel flex items-center gap-4">
          <div className="bg-emerald-500/20 p-4 rounded-xl">
            <Activity className="text-emerald-400" size={24} />
          </div>
          <div>
            <p className="text-sm text-slate-400">Uptime</p>
            <p className="text-2xl font-bold">{stats?.uptime_percent.toFixed(2)}%</p>
          </div>
        </div>

        <div className="glass-panel flex items-center gap-4">
          <div className="bg-blue-500/20 p-4 rounded-xl">
            <Clock className="text-blue-400" size={24} />
          </div>
          <div>
            <p className="text-sm text-slate-400">Avg Response</p>
            <p className="text-2xl font-bold">{stats?.avg_response_time.toFixed(0)} ms</p>
          </div>
        </div>

        <div className="glass-panel flex items-center gap-4">
          <div className="bg-purple-500/20 p-4 rounded-xl">
            <ServerCrash className="text-purple-400" size={24} />
          </div>
          <div>
            <p className="text-sm text-slate-400">Total Checks</p>
            <p className="text-2xl font-bold">{stats?.total_checks}</p>
          </div>
        </div>
      </div>

      <div className="glass-panel">
        <h2 className="text-xl font-bold mb-6">Recent Checks (Last 100)</h2>
        
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="border-b border-white/10">
                <th className="pb-3 text-sm font-medium text-slate-400">Time</th>
                <th className="pb-3 text-sm font-medium text-slate-400">Status</th>
                <th className="pb-3 text-sm font-medium text-slate-400">Response (ms)</th>
              </tr>
            </thead>
            <tbody>
              {checks.map(c => (
                <tr key={c.id} className="border-b border-white/5 last:border-0 hover:bg-white/5 transition-colors">
                  <td className="py-3 text-sm">{new Date(c.created_at).toLocaleString()}</td>
                  <td className="py-3">
                    {c.status === 'UP' ? (
                      <span className="inline-flex items-center gap-1.5 text-emerald-400 text-sm font-medium">
                        <CheckCircle2 size={16} /> UP
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 text-red-400 text-sm font-medium">
                        <ServerCrash size={16} /> DOWN
                      </span>
                    )}
                  </td>
                  <td className="py-3 text-sm text-slate-300">{c.response_time}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {checks.length === 0 && <p className="text-center py-4 text-slate-400">No checks recorded yet.</p>}
        </div>
      </div>
    </div>
  );
};

export default ServiceDetails;
