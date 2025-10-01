import React from 'react';

const TestApp: React.FC = () => {
  return (
    <div className="min-h-screen bg-slate-900 text-white flex flex-col items-center justify-center gap-6">
      <div className="text-3xl font-bold tracking-wide">Test - Can you see this?</div>
      <button className="px-6 py-3 rounded-lg bg-sky-500 hover:bg-sky-400 transition focus:ring-4 focus:ring-sky-300">
        Tailwind Button
      </button>
      <p className="text-sm text-sky-200 max-w-md text-center">
        If this card is not dark blue or the button is missing its hover effect, Tailwind CSS is not loading.
      </p>
    </div>
  );
};

export default TestApp;
