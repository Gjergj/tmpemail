import { Link } from 'react-router-dom';
import { Helmet } from 'react-helmet-async';
import { blogPosts } from '../data/blogPosts';

export function Blog() {
  return (
    <>
      <Helmet>
        <title>Blog - tmpemail.xyz</title>
        <meta
          name="description"
          content="Tips, guides, and insights about email privacy. Learn how to protect your inbox and stay safe online."
        />
        <meta property="og:title" content="Blog - tmpemail.xyz" />
        <meta
          property="og:description"
          content="Tips, guides, and insights about email privacy."
        />
        <meta property="og:type" content="website" />
        <meta property="og:url" content="https://tmpemail.xyz/blog" />
        <meta name="twitter:card" content="summary" />
        <meta name="twitter:title" content="Blog - tmpemail.xyz" />
        <meta
          name="twitter:description"
          content="Tips, guides, and insights about email privacy."
        />
        <link rel="canonical" href="https://tmpemail.xyz/blog" />
      </Helmet>

      <header className="text-center mb-10">
        <h1 className="text-3xl font-bold text-slate-800 mb-2">Blog</h1>
        <p className="text-slate-500">
          Tips, guides, and insights about email privacy
        </p>
      </header>

      <div className="space-y-6">
        {blogPosts.map((post) => (
          <Link
            key={post.slug}
            to={`/blog/${post.slug}`}
            className="block bg-white rounded-lg p-6 shadow-sm border border-slate-200 hover:border-slate-300 hover:shadow-md transition-all"
          >
            <article>
              <div className="flex items-center gap-3 text-sm text-slate-400 mb-2">
                <time dateTime={post.date}>
                  {new Date(post.date).toLocaleDateString('en-US', {
                    month: 'long',
                    day: 'numeric',
                    year: 'numeric',
                  })}
                </time>
                <span>•</span>
                <span>{post.readTime}</span>
              </div>
              <h2 className="text-xl font-semibold text-slate-800 mb-2 group-hover:text-slate-600">
                {post.title}
              </h2>
              <p className="text-slate-600 leading-relaxed">{post.excerpt}</p>
            </article>
          </Link>
        ))}
      </div>
    </>
  );
}
