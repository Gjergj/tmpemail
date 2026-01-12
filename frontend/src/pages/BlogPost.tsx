import { useParams, Link } from 'react-router-dom';
import { Helmet } from 'react-helmet-async';
import { blogPosts, blogContent } from '../data/blogPosts';

export function BlogPost() {
  const { slug } = useParams<{ slug: string }>();
  const post = blogPosts.find((p) => p.slug === slug);
  const content = slug ? blogContent[slug] : null;

  if (!post || !content) {
    return (
      <>
        <Helmet>
          <title>Post Not Found - tmpemail.xyz</title>
          <meta name="robots" content="noindex" />
        </Helmet>
        <div className="text-center py-10">
          <h1 className="text-2xl font-bold text-slate-800 mb-4">
            Post Not Found
          </h1>
          <p className="text-slate-500 mb-6">
            The blog post you're looking for doesn't exist.
          </p>
          <Link
            to="/blog"
            className="text-slate-600 hover:text-slate-800 underline"
          >
            ← Back to Blog
          </Link>
        </div>
      </>
    );
  }

  return (
    <article>
      <Helmet>
        <title>{post.title} - tmpemail.xyz</title>
        <meta name="description" content={post.excerpt} />
        <meta property="og:title" content={`${post.title} - tmpemail.xyz`} />
        <meta property="og:description" content={post.excerpt} />
        <meta property="og:type" content="article" />
        <meta property="og:url" content={`https://tmpemail.xyz/blog/${slug}`} />
        <meta property="article:published_time" content={post.date} />
        <meta name="twitter:card" content="summary" />
        <meta name="twitter:title" content={`${post.title} - tmpemail.xyz`} />
        <meta name="twitter:description" content={post.excerpt} />
        <link rel="canonical" href={`https://tmpemail.xyz/blog/${slug}`} />
      </Helmet>
      <Link
        to="/blog"
        className="inline-flex items-center text-slate-500 hover:text-slate-700 mb-6 text-sm"
      >
        <svg
          className="w-4 h-4 mr-1"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M15 19l-7-7 7-7"
          />
        </svg>
        Back to Blog
      </Link>

      <header className="mb-8">
        <div className="flex items-center gap-3 text-sm text-slate-400 mb-3">
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
        <h1 className="text-3xl font-bold text-slate-800">{post.title}</h1>
      </header>

      <div className="prose prose-slate max-w-none">
        {content.split('\n').map((line, index) => {
          if (line.startsWith('## ')) {
            return (
              <h2
                key={index}
                className="text-xl font-semibold text-slate-800 mt-8 mb-4"
              >
                {line.replace('## ', '')}
              </h2>
            );
          }
          if (line.startsWith('### ')) {
            return (
              <h3
                key={index}
                className="text-lg font-semibold text-slate-700 mt-6 mb-3"
              >
                {line.replace('### ', '')}
              </h3>
            );
          }
          if (line.startsWith('- ')) {
            return (
              <li key={index} className="text-slate-600 ml-4 mb-1">
                {line.replace('- ', '')}
              </li>
            );
          }
          if (line.startsWith('---')) {
            return <hr key={index} className="my-8 border-slate-200" />;
          }
          if (line.startsWith('**') && line.endsWith('**')) {
            return (
              <p key={index} className="font-semibold text-slate-700 my-4">
                {line.replace(/\*\*/g, '')}
              </p>
            );
          }
          if (line.trim()) {
            // Handle inline bold text
            const parts = line.split(/(\*\*[^*]+\*\*)/g);
            return (
              <p key={index} className="text-slate-600 leading-relaxed mb-4">
                {parts.map((part, i) =>
                  part.startsWith('**') && part.endsWith('**') ? (
                    <strong key={i} className="font-semibold text-slate-700">
                      {part.replace(/\*\*/g, '')}
                    </strong>
                  ) : (
                    part
                  )
                )}
              </p>
            );
          }
          return null;
        })}
      </div>

      <div className="mt-12 pt-8 border-t border-slate-200">
        <Link
          to="/"
          className="inline-flex items-center justify-center w-full px-5 py-3 bg-slate-500 text-white rounded-md font-medium hover:bg-slate-600 transition-colors"
        >
          Get Your Temporary Email →
        </Link>
      </div>
    </article>
  );
}
