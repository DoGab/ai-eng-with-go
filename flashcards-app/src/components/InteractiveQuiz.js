import React, { useState, useRef, useEffect } from 'react';
import { AcademicCapIcon, ChatBubbleLeftRightIcon, ChartBarIcon, CheckCircleIcon, ArrowLeftIcon, ArrowRightIcon } from '@heroicons/react/24/outline';
import { interactiveQuizApi, notesApi } from '../api/flashcardsApi';
import toast from 'react-hot-toast';
import ReactMarkdown from 'react-markdown';

const InteractiveQuiz = () => {
  const [quizState, setQuizState] = useState({
    step: 'configure', // 'configure' | 'ranking' | 'quiz' | 'complete'
    configuration: null, // { note_ids, question_count, topics, message }
    rankedNotes: [],
    selectedNote: null, // The actual note object with content
    currentQuestionIndex: 0,
    quizResults: [],
    currentQuizMessages: []
  });

  const [configMessages, setConfigMessages] = useState([]);
  const [currentConfigMessage, setCurrentConfigMessage] = useState('');
  const [loading, setLoading] = useState(false);

  // Refs for auto-scrolling
  const configMessagesEndRef = useRef(null);
  const quizMessagesEndRef = useRef(null);

  // Auto-scroll to bottom when messages update
  useEffect(() => {
    if (configMessagesEndRef.current) {
      configMessagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [configMessages]);

  useEffect(() => {
    if (quizMessagesEndRef.current) {
      quizMessagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [quizState.currentQuizMessages]);

  const steps = [
    { id: 'configure', name: 'Configure', icon: ChatBubbleLeftRightIcon },
    { id: 'ranking', name: 'Select Note', icon: ChartBarIcon },
    { id: 'quiz', name: 'Quiz', icon: AcademicCapIcon },
    { id: 'complete', name: 'Complete', icon: CheckCircleIcon }
  ];

  const getCurrentStepIndex = () => {
    return steps.findIndex(step => step.id === quizState.step);
  };

  const resetQuiz = () => {
    setQuizState({
      step: 'configure',
      configuration: null,
      rankedNotes: [],
      selectedNote: null,
      currentQuestionIndex: 0,
      quizResults: [],
      currentQuizMessages: []
    });
    setConfigMessages([]);
    setCurrentConfigMessage('');
  };

  const startConfiguration = async () => {
    if (!currentConfigMessage.trim()) {
      toast.error('Please enter your quiz preferences');
      return;
    }

    const userMessage = {
      role: 'user',
      content: currentConfigMessage
    };

    const updatedMessages = [...configMessages, userMessage];
    setConfigMessages(updatedMessages);
    setCurrentConfigMessage('');
    setLoading(true);

    try {
      const response = await interactiveQuizApi.configure(updatedMessages);
      const aiResponse = {
        role: 'assistant',
        content: response.data.message
      };

      setConfigMessages([...updatedMessages, aiResponse]);

      if (response.data.type === 'configure') {
        // Configuration is complete
        setQuizState(prev => ({
          ...prev,
          configuration: response.data.config,
          step: 'ranking'
        }));
        toast.success('Quiz configured! Now selecting the best note...');
        // Automatically move to ranking
        await rankNotes(response.data.config);
      }
    } catch (error) {
      toast.error('Failed to configure quiz');
      console.error('Configuration error:', error);
    } finally {
      setLoading(false);
    }
  };

  const rankNotes = async (config) => {
    setLoading(true);
    try {
      const response = await interactiveQuizApi.rank(config.note_ids, config.topic.split(', '));
      const rankedNotes = response.data.ranked_notes;
      
      if (rankedNotes.length > 0) {
        // Fetch the actual note content for the top-ranked note
        const topNoteId = rankedNotes[0].note_id;
        const noteResponse = await notesApi.getById(topNoteId);
        
        setQuizState(prev => ({
          ...prev,
          rankedNotes: rankedNotes,
          selectedNote: noteResponse.data,
          step: 'ranking'
        }));
      } else {
        setQuizState(prev => ({
          ...prev,
          rankedNotes: rankedNotes,
          step: 'ranking'
        }));
      }
    } catch (error) {
      toast.error('Failed to rank notes');
      console.error('Ranking error:', error);
    } finally {
      setLoading(false);
    }
  };

  const startQuiz = async () => {
    setQuizState(prev => ({
      ...prev,
      step: 'quiz',
      currentQuestionIndex: 0,
      currentQuizMessages: []
    }));

    // Automatically start the quiz with the first question
    await autoStartNextQuestion();
  };

  const autoStartNextQuestion = async () => {
    if (!quizState.configuration || quizState.rankedNotes.length === 0) return;

    const topNote = quizState.rankedNotes[0];
    const topics = quizState.configuration.topic.split(', ');

    setLoading(true);
    try {
      // Generate question with empty messages (no user message added)
      const response = await interactiveQuizApi.conduct([topNote.note_id], topics, []);
      
      const aiResponse = {
        role: 'assistant',
        content: response.data.message
      };

      setQuizState(prev => ({
        ...prev,
        currentQuizMessages: [aiResponse]
      }));
    } catch (error) {
      toast.error('Failed to generate question');
      console.error('Auto question generation error:', error);
    } finally {
      setLoading(false);
    }
  };

  const sendQuizMessage = async (message) => {
    if (!quizState.configuration || quizState.rankedNotes.length === 0) return;

    const topNote = quizState.rankedNotes[0];
    const topics = quizState.configuration.topic.split(', ');
    
    let updatedMessages;
    if (message.trim() === '' && quizState.currentQuizMessages.length === 0) {
      // Initial question generation with empty messages
      updatedMessages = [];
    } else {
      // Regular user message
      const userMessage = { role: 'user', content: message };
      updatedMessages = [...quizState.currentQuizMessages, userMessage];
      
      setQuizState(prev => ({
        ...prev,
        currentQuizMessages: updatedMessages
      }));
    }

    setLoading(true);
    try {
      const response = await interactiveQuizApi.conduct([topNote.note_id], topics, updatedMessages);
      
      const aiResponse = {
        role: 'assistant',
        content: response.data.message
      };

      const finalMessages = [...updatedMessages, aiResponse];
      
      setQuizState(prev => ({
        ...prev,
        currentQuizMessages: finalMessages
      }));

      if (response.data.type === 'evaluate') {
        // Quiz question completed
        const result = {
          questionNumber: quizState.currentQuestionIndex + 1,
          evaluation: response.data.evaluation,
          messages: finalMessages
        };

        setQuizState(prev => ({
          ...prev,
          quizResults: [...prev.quizResults, result],
          currentQuestionIndex: prev.currentQuestionIndex + 1,
          currentQuizMessages: []
        }));

        // Check if quiz is complete
        if (quizState.currentQuestionIndex + 1 >= quizState.configuration.question_count) {
          setQuizState(prev => ({
            ...prev,
            step: 'complete'
          }));
          toast.success('Quiz completed!');
        } else {
          toast.success(`Question ${quizState.currentQuestionIndex + 1} completed!`);
          // Automatically start the next question
          setTimeout(async () => {
            await autoStartNextQuestion();
          }, 500); // Small delay to let the state update
        }
      }
    } catch (error) {
      toast.error('Failed to process quiz message');
      console.error('Quiz conduct error:', error);
    } finally {
      setLoading(false);
    }
  };

  const renderStepIndicator = () => {
    const currentIndex = getCurrentStepIndex();
    
    return (
      <div className="flex items-center justify-center mb-8">
        {steps.map((step, index) => {
          const Icon = step.icon;
          const isActive = index === currentIndex;
          const isCompleted = index < currentIndex;
          
          return (
            <div key={step.id} className="flex items-center">
              <div className={`flex items-center justify-center w-10 h-10 rounded-full border-2 ${
                isActive ? 'border-green-500 bg-green-500 text-white' :
                isCompleted ? 'border-green-500 bg-green-500 text-white' :
                'border-gray-300 text-gray-400'
              }`}>
                <Icon className="h-5 w-5" />
              </div>
              <span className={`ml-2 text-sm font-medium ${
                isActive ? 'text-green-600' :
                isCompleted ? 'text-green-600' :
                'text-gray-400'
              }`}>
                {step.name}
              </span>
              {index < steps.length - 1 && (
                <ArrowRightIcon className="h-4 w-4 text-gray-400 mx-4" />
              )}
            </div>
          );
        })}
      </div>
    );
  };

  const renderConfigurationStep = () => (
    <div className="space-y-6">
      <div className="text-center">
        <h3 className="text-xl font-semibold text-gray-800 mb-2">Let's Configure Your Quiz</h3>
        <p className="text-gray-600">Tell me what you'd like to study and I'll help you create the perfect quiz!</p>
      </div>

      {/* Configuration Messages */}
      {configMessages.length > 0 && (
        <div className="space-y-4 max-h-64 overflow-y-auto">
          {configMessages.map((message, index) => (
            <div
              key={index}
              className={`p-4 rounded-lg ${
                message.role === 'user'
                  ? 'bg-blue-50 border-l-4 border-blue-500 ml-8'
                  : 'bg-gray-50 border-l-4 border-gray-500 mr-8'
              }`}
            >
              <div className="flex items-center gap-2 mb-2">
                <span className={`text-sm font-medium ${
                  message.role === 'user' ? 'text-blue-700' : 'text-gray-700'
                }`}>
                  {message.role === 'user' ? '🧑 You' : '🤖 AI Assistant'}
                </span>
              </div>
              <div className="text-gray-800 prose prose-sm max-w-none">
                <ReactMarkdown>{message.content}</ReactMarkdown>
              </div>
            </div>
          ))}
          <div ref={configMessagesEndRef} />
        </div>
      )}

      {/* Input Form */}
      <form onSubmit={(e) => { e.preventDefault(); startConfiguration(); }} className="flex gap-4">
        <input
          type="text"
          value={currentConfigMessage}
          onChange={(e) => setCurrentConfigMessage(e.target.value)}
          placeholder={configMessages.length === 0 ? 
            "I want to study database performance..." : 
            "Continue the conversation..."
          }
          className="flex-1 p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-transparent"
          disabled={loading}
        />
        <button
          type="submit"
          disabled={!currentConfigMessage.trim() || loading}
          className="bg-green-600 hover:bg-green-700 disabled:bg-gray-400 text-white px-6 py-3 rounded-lg transition-colors"
        >
          {loading ? 'Processing...' : 'Send'}
        </button>
      </form>
    </div>
  );

  const truncateText = (text, maxLength = 400) => {
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
  };

  const renderRankingStep = () => {
    if (!quizState.configuration || quizState.rankedNotes.length === 0 || !quizState.selectedNote) {
      return (
        <div className="text-center py-8">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-green-600 mx-auto"></div>
          <p className="mt-4 text-gray-600">Selecting the best note for your quiz...</p>
        </div>
      );
    }

    const topNote = quizState.rankedNotes[0];
    const selectedNote = quizState.selectedNote;

    return (
      <div className="space-y-6">
        <div className="text-center">
          <h3 className="text-xl font-semibold text-gray-800 mb-2">Perfect! I've Selected Your Study Material</h3>
          <p className="text-gray-600">Based on your preferences, here's the best note to quiz you on:</p>
        </div>

        <div className="bg-green-50 border border-green-200 rounded-lg p-6">
          <div className="flex items-center justify-between mb-4">
            <h4 className="text-lg font-medium text-green-800">Selected Note (Relevance: {(topNote.score * 100).toFixed(0)}%)</h4>
            <span className="text-sm text-green-600 font-medium">Note ID: {topNote.note_id}</span>
          </div>
          
          {/* Note Content Preview */}
          <div className="bg-white border border-green-300 rounded-lg p-4 mb-4">
            <h5 className="text-sm font-medium text-gray-700 mb-2">📝 Note Content:</h5>
            <div className="text-gray-800 prose prose-sm max-w-none">
              <ReactMarkdown>{truncateText(selectedNote.content)}</ReactMarkdown>
            </div>
            {selectedNote.content.length > 400 && (
              <p className="text-sm text-gray-500 mt-2 italic">Content truncated for preview...</p>
            )}
          </div>

          <div className="text-gray-700 prose prose-sm max-w-none">
            <p className="italic">This note will be used to generate quiz questions about: <strong>{quizState.configuration.topic}</strong></p>
          </div>
        </div>


        <div className="flex gap-4 justify-center">
          <button
            onClick={resetQuiz}
            className="bg-gray-600 hover:bg-gray-700 text-white px-6 py-3 rounded-lg flex items-center gap-2 transition-colors"
          >
            <ArrowLeftIcon className="h-5 w-5" />
            Reconfigure
          </button>
          <button
            onClick={startQuiz}
            className="bg-green-600 hover:bg-green-700 text-white px-6 py-3 rounded-lg flex items-center gap-2 transition-colors"
          >
            Start Quiz
            <ArrowRightIcon className="h-5 w-5" />
          </button>
        </div>
      </div>
    );
  };

  const renderQuizStep = () => {
    const isFirstQuestion = quizState.currentQuizMessages.length === 0;
    
    return (
      <div className="space-y-6">
        <div className="text-center">
          <h3 className="text-xl font-semibold text-gray-800 mb-2">
            Question {quizState.currentQuestionIndex + 1} of {quizState.configuration.question_count}
          </h3>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div 
              className="bg-green-600 h-2 rounded-full transition-all duration-300"
              style={{ width: `${((quizState.currentQuestionIndex) / quizState.configuration.question_count) * 100}%` }}
            ></div>
          </div>
        </div>

        {/* Quiz Messages */}
        <div className="space-y-4 max-h-64 overflow-y-auto">
          {quizState.currentQuizMessages.map((message, index) => (
            <div
              key={index}
              className={`p-4 rounded-lg ${
                message.role === 'user'
                  ? 'bg-blue-50 border-l-4 border-blue-500 ml-8'
                  : 'bg-gray-50 border-l-4 border-gray-500 mr-8'
              }`}
            >
              <div className="flex items-center gap-2 mb-2">
                <span className={`text-sm font-medium ${
                  message.role === 'user' ? 'text-blue-700' : 'text-gray-700'
                }`}>
                  {message.role === 'user' ? '🧑 You' : '🤖 AI Tutor'}
                </span>
              </div>
              <div className="text-gray-800 prose prose-sm max-w-none">
                <ReactMarkdown>{message.content}</ReactMarkdown>
              </div>
            </div>
          ))}
          
          {/* Loading state for initial question generation */}
          {loading && isFirstQuestion && (
            <div className="bg-gray-50 border-l-4 border-gray-500 mr-8 p-4 rounded-lg">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-sm font-medium text-gray-700">🤖 AI Tutor</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-gray-600"></div>
                <span className="text-gray-600">Generating your question...</span>
              </div>
            </div>
          )}
          
          <div ref={quizMessagesEndRef} />
        </div>

        {/* Input Form - Only show if there are messages or not loading */}
        {(quizState.currentQuizMessages.length > 0 || !loading) && (
          <form onSubmit={(e) => { 
            e.preventDefault(); 
            const formData = new FormData(e.target);
            const message = formData.get('message');
            if (message.trim()) {
              sendQuizMessage(message);
              e.target.reset();
            }
          }} className="flex gap-4">
            <input
              name="message"
              type="text"
              placeholder="Type your answer or ask for clarification..."
              className="flex-1 p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-transparent"
              disabled={loading}
            />
            <button
              type="submit"
              disabled={loading}
              className="bg-green-600 hover:bg-green-700 disabled:bg-gray-400 text-white px-6 py-3 rounded-lg transition-colors"
            >
              {loading ? 'Processing...' : 'Send'}
            </button>
          </form>
        )}
      </div>
    );
  };

  const renderCompleteStep = () => (
    <div className="space-y-6 text-center">
      <div>
        <CheckCircleIcon className="h-16 w-16 text-green-500 mx-auto mb-4" />
        <h3 className="text-2xl font-semibold text-gray-800 mb-2">Quiz Complete! 🎉</h3>
        <p className="text-gray-600">Great job! Here's how you performed:</p>
      </div>

      <div className="space-y-4">
        {quizState.quizResults.map((result, index) => (
          <div key={index} className="bg-gray-50 border rounded-lg p-4 text-left">
            <div className="flex items-center justify-between mb-2">
              <h4 className="font-medium text-gray-800">Question {result.questionNumber}</h4>
              <span className={`px-3 py-1 rounded-full text-sm font-medium ${
                result.evaluation.correct 
                  ? 'bg-green-100 text-green-800' 
                  : 'bg-red-100 text-red-800'
              }`}>
                {result.evaluation.correct ? '✓ Correct' : '✗ Incorrect'}
              </span>
            </div>
            <div className="text-gray-700 prose prose-sm max-w-none">
              <ReactMarkdown>{result.evaluation.feedback}</ReactMarkdown>
            </div>
          </div>
        ))}
      </div>

      <div className="flex gap-4 justify-center">
        <button
          onClick={resetQuiz}
          className="bg-green-600 hover:bg-green-700 text-white px-8 py-3 rounded-lg transition-colors"
        >
          Start New Quiz
        </button>
      </div>
    </div>
  );

  const renderCurrentStep = () => {
    switch (quizState.step) {
      case 'configure':
        return renderConfigurationStep();
      case 'ranking':
        return renderRankingStep();
      case 'quiz':
        return renderQuizStep();
      case 'complete':
        return renderCompleteStep();
      default:
        return renderConfigurationStep();
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="bg-white rounded-lg shadow-lg p-6">
        <h2 className="text-2xl font-bold text-gray-800 mb-6">🎓 Interactive Quiz</h2>
        
        {renderStepIndicator()}
        {renderCurrentStep()}
      </div>
    </div>
  );
};

export default InteractiveQuiz;
