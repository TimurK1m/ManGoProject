import React from 'react';
import { Activity, LogOut } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { Link } from 'react-router-dom';

const Navbar: React.FC = () => {
  const { logout } = useAuth();

  return (
    <nav className="glass-panel rounded-none border-t-0 border-x-0 border-b border-white/10 px-8 py-4 flex items-center justify-between sticky top-0 z-50">
      <Link to="/" className="flex items-center gap-2 text-xl font-bold text-white hover:text-blue-400 transition-colors">
        <Activity className="text-blue-500" />
        ManGo
      </Link>
      
      <button onClick={logout} className="btn-secondary !py-1.5 !px-3 text-sm">
        <LogOut size={16} />
        Logout
      </button>
    </nav>
  );
};

export default Navbar;
