import React, { useEffect, useState } from 'react';
import { api } from '../api';
import { Link } from 'react-router-dom';
import { Plus, Server, Activity, Trash2, ArrowRight } from 'lucide-react';

interface Service {
  id: number;
  url: string;
  check_interval: number;
  last_status: { String: string; Valid: boolean };
}

const Dashboard: React.FC = () => {
  const [services, setServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [newUrl, setNewUrl] = useState('');
  const [newInterval, setNewInterval] = useState('30');
  const [addError, setAddError] = useState('');

  const fetchServices = async () => {
    try {
      const res = await api.get('/services');
      setServices(res.data.services || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServices();
    const interval = setInterval(fetchServices, 10000); // Poll every 10s
    return () => clearInterval(interval);
  }, []);

  const handleAddService = async (e: React.FormEvent) => {
    e.preventDefault();
    setAddError('');
    try {
      await api.post('/services', { 
        url: newUrl,
        check_interval: parseInt(newInterval, 10)
      });
      setNewUrl('');
      setNewInterval('30');
      setShowAdd(false);
      fetchServices();
    } catch (err: any) {
      setAddError(err.response?.data?.error || 'Failed to add service');
    }
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure?')) return;
    try {
      await api.delete(`/services/${id}`);
      fetchServices();
    } catch (err) {
      console.error(err);
    }
  };

  if (loading) {
    return <div className="text-center py-20 text-slate-400">Loading services...</div>;
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-8">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="text-slate-400">Monitor your web services in real-time</p>
        </div>
        <button onClick={() => setShowAdd(!showAdd)} className="btn-primary">
          <Plus size={20} />
          Add Service
        </button>
      </div>

      {showAdd && (
        <div className="glass-panel mb-8 animate-[pulse_0.2s_ease-out_1]">
          <h2 className="text-lg mb-4">Add New Service</h2>
          <form onSubmit={handleAddService} className="flex flex-col md:flex-row gap-4 items-start">
            <div className="flex-1 w-full">
              <input
                type="url"
                required
                className="input-field mb-0"
                placeholder="https://example.com"
                value={newUrl}
                onChange={(e) => setNewUrl(e.target.value)}
              />
              {addError && <p className="text-red-400 text-sm mt-1">{addError}</p>}
            </div>
            <div className="w-32">
              <input
                type="number"
                required
                min="10"
                className="input-field mb-0"
                placeholder="30"
                value={newInterval}
                onChange={(e) => setNewInterval(e.target.value)}
              />
            </div>
            <div className="flex gap-2">
              <button type="submit" className="btn-primary">Save</button>
              <button type="button" onClick={() => setShowAdd(false)} className="btn-secondary">Cancel</button>
            </div>
          </form>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {services.length === 0 ? (
          <div className="col-span-full text-center py-12 glass-panel text-slate-400">
            No services registered yet. Click "Add Service" to start monitoring.
          </div>
        ) : (
          services.map(s => {
            const status = s.last_status.Valid ? s.last_status.String : 'PENDING';
            const statusClass = status === 'UP' ? 'badge-up' : status === 'DOWN' ? 'badge-down' : 'bg-slate-700 text-slate-300';
            
            return (
              <div key={s.id} className="glass-panel hover:-translate-y-1 transition-transform relative group">
                <div className="flex justify-between items-start mb-4">
                  <div className="bg-blue-500/20 p-2 rounded-lg">
                    <Server size={20} className="text-blue-400" />
                  </div>
                  <span className={`badge ${statusClass}`}>
                    {status === 'UP' && <span className="w-2 h-2 rounded-full bg-emerald-400 mr-1.5 animate-pulse"></span>}
                    {status === 'DOWN' && <span className="w-2 h-2 rounded-full bg-red-400 mr-1.5"></span>}
                    {status}
                  </span>
                </div>
                
                <h3 className="text-lg font-medium truncate mb-1" title={s.url}>{s.url}</h3>
                <p className="text-xs text-slate-400 mb-2">Checks every {s.check_interval}s</p>
                
                <div className="mt-4 flex gap-2">
                  <Link to={`/services/${s.id}`} className="btn-secondary flex-1 !text-sm">
                    View Stats <ArrowRight size={14} />
                  </Link>
                  <button onClick={() => handleDelete(s.id)} className="btn-secondary !text-red-400 hover:!bg-red-500/10 px-3" title="Delete">
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

export default Dashboard;
