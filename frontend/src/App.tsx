import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import { Home } from './pages/Home';
import { Blog } from './pages/Blog';
import { BlogPost } from './pages/BlogPost';

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-slate-50 p-5">
        <div className="max-w-3xl mx-auto">
          <nav className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-6">
              <NavLink
                to="/"
                className={({ isActive }) =>
                  `text-sm font-medium transition-colors ${isActive
                    ? 'text-slate-800'
                    : 'text-slate-400 hover:text-slate-600'
                  }`
                }
              >
                Home
              </NavLink>
              <NavLink
                to="/blog"
                className={({ isActive }) =>
                  `text-sm font-medium transition-colors ${isActive
                    ? 'text-slate-800'
                    : 'text-slate-400 hover:text-slate-600'
                  }`
                }
              >
                Blog
              </NavLink>
            </div>
            <a
              href="https://x.com/gjergjiramku"
              target="_blank"
              rel="noopener noreferrer"
              className="text-slate-400 hover:text-slate-600 transition-colors"
              aria-label="Follow on X"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="currentColor"
              >
                <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
              </svg>
            </a>
          </nav>

          <Routes>
            <Route path="/" element={<Home />} />
            <Route path="/blog" element={<Blog />} />
            <Route path="/blog/:slug" element={<BlogPost />} />
          </Routes>
        </div>
      </div>
    </BrowserRouter>
  );
}

export default App;
