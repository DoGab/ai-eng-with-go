import React from 'react';
import { BookOpenIcon, SparklesIcon, AcademicCapIcon, HeartIcon } from '@heroicons/react/24/outline';

const Navigation = ({ activeTab, setActiveTab }) => {
  const tabs = [
    { id: 'notes', name: 'Notes', icon: BookOpenIcon },
    { id: 'quiz', name: 'AI Quiz', icon: SparklesIcon },
    { id: 'interactive', name: 'Interactive Quiz', icon: AcademicCapIcon },
  ];

  return (
    <nav className="bg-white shadow-sm border-b border-gray-200">
      <div className="max-w-4xl mx-auto px-6">
        <div className="flex justify-between items-center py-4">
          <div className="flex items-center gap-2">
            <BookOpenIcon className="h-8 w-8 text-blue-600" />
            <h1 className="text-xl font-bold text-gray-800">Flashcards App</h1>
          </div>
          
          <div className="flex space-x-1">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-colors ${
                    activeTab === tab.id
                      ? 'bg-blue-100 text-blue-700'
                      : 'text-gray-600 hover:text-gray-800 hover:bg-gray-100'
                  }`}
                >
                  <Icon className="h-5 w-5" />
                  {tab.name}
                </button>
              );
            })}
          </div>
          
          <div className="flex items-center gap-2 text-sm text-gray-500">
            <span>Made with</span>
            <HeartIcon className="h-4 w-4 text-red-500" />
            <span>& AI</span>
          </div>
        </div>
      </div>
    </nav>
  );
};

export default Navigation;