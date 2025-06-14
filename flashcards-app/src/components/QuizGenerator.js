import React, { useState, useEffect } from 'react';
import { SparklesIcon, PaperAirplaneIcon } from '@heroicons/react/24/outline';
import { quizApi, notesApi } from '../api/flashcardsApi';
import toast from 'react-hot-toast';
import ReactMarkdown from 'react-markdown';

const QuizGenerator = () => {
  const [notes, setNotes] = useState([]);
  const [selectedNotes, setSelectedNotes] = useState([]);
  const [messages, setMessages] = useState([]);
  const [currentMessage, setCurrentMessage] = useState('');
  const [loading, setLoading] = useState(false);
  const [loadingNotes, setLoadingNotes] = useState(true);
  const [isStreaming, setIsStreaming] = useState(false);
  const [streamingContent, setStreamingContent] = useState('');

  const truncateText = (text, maxLength = 200) => {
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
  };

  useEffect(() => {
    fetchNotes();
  }, []);

  const fetchNotes = async () => {
    try {
      const response = await notesApi.getAll();
      setNotes(response.data);
    } catch (error) {
      toast.error('Failed to fetch notes');
      console.error('Error fetching notes:', error);
    } finally {
      setLoadingNotes(false);
    }
  };

  const handleNoteSelection = (noteId) => {
    setSelectedNotes(prev => 
      prev.includes(noteId) 
        ? prev.filter(id => id !== noteId)
        : [...prev, noteId]
    );
  };

  const generateInitialQuiz = async () => {
    if (selectedNotes.length === 0) {
      toast.error('Please select at least one note to generate a quiz');
      return;
    }

    setLoading(true);
    setIsStreaming(true);
    setStreamingContent('');
    
    try {
      const selectedNoteIDs = selectedNotes;
      console.log(selectedNoteIDs);
      const initialMessages = [];

      await quizApi.generateStream(
        selectedNoteIDs, 
        initialMessages,
        (token, accumulatedContent) => {
          setStreamingContent(accumulatedContent);
        },
        (finalContent) => {
          const newMessage = {
            role: 'assistant',
            content: finalContent
          };
          setMessages([newMessage]);
          setStreamingContent('');
          setIsStreaming(false);
          setLoading(false);
          toast.success('Quiz generated successfully!');
        },
        (error) => {
          console.error('Streaming error:', error);
          setIsStreaming(false);
          setLoading(false);
          toast.error('Failed to generate quiz');
        }
      );
    } catch (error) {
      toast.error('Failed to generate quiz');
      console.error('Error generating quiz:', error);
      setIsStreaming(false);
      setLoading(false);
    }
  };

  const sendMessage = async (e) => {
    e.preventDefault();
    if (!currentMessage.trim()) return;

    const selectedNoteIDs = selectedNotes;
    const newUserMessage = {
      role: 'user',
      content: currentMessage
    };

    const updatedMessages = [...messages, newUserMessage];
    setMessages(updatedMessages);
    setCurrentMessage('');
    setLoading(true);
    setIsStreaming(true);
    setStreamingContent('');

    try {
      await quizApi.generateStream(
        selectedNoteIDs, 
        updatedMessages,
        (token, accumulatedContent) => {
          setStreamingContent(accumulatedContent);
        },
        (finalContent) => {
          const newAssistantMessage = {
            role: 'assistant',
            content: finalContent
          };
          setMessages([...updatedMessages, newAssistantMessage]);
          setStreamingContent('');
          setIsStreaming(false);
          setLoading(false);
        },
        (error) => {
          console.error('Streaming error:', error);
          setIsStreaming(false);
          setLoading(false);
          toast.error('Failed to send message');
        }
      );
    } catch (error) {
      toast.error('Failed to send message');
      console.error('Error sending message:', error);
      setIsStreaming(false);
      setLoading(false);
    }
  };

  const clearConversation = () => {
    setMessages([]);
    setSelectedNotes([]);
    setStreamingContent('');
    setIsStreaming(false);
  };

  if (loadingNotes) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-600"></div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="bg-white rounded-lg shadow-lg p-6">
        <h2 className="text-2xl font-bold text-gray-800 mb-6">🧠 AI Quiz Generator</h2>
        
        {/* Notes Selection */}
        <div className="mb-6">
          <h3 className="text-lg font-semibold text-gray-700 mb-3">Select Notes for Quiz Generation</h3>
          {notes.length === 0 ? (
            <p className="text-gray-500 text-center py-4">
              No notes available. Create some notes first to generate quizzes.
            </p>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
              {notes.map((note) => (
                <div
                  key={note.id}
                  className={`p-4 border rounded-lg cursor-pointer transition-all ${
                    selectedNotes.includes(note.id)
                      ? 'border-purple-500 bg-purple-50'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                  onClick={() => handleNoteSelection(note.id)}
                >
                  <div className="text-sm text-gray-600 prose prose-xs max-w-none">
                    <ReactMarkdown>
                      {truncateText(note.content)}
                    </ReactMarkdown>
                  </div>
                  <p className="text-xs text-gray-400 mt-2">
                    {new Date(note.createdAt).toLocaleDateString()}
                  </p>
                </div>
              ))}
            </div>
          )}
          
          <div className="flex gap-4">
            <button
              onClick={generateInitialQuiz}
              disabled={selectedNotes.length === 0 || loading || isStreaming}
              className="bg-purple-600 hover:bg-purple-700 disabled:bg-gray-400 text-white px-6 py-2 rounded-lg flex items-center gap-2 transition-colors"
            >
              <SparklesIcon className="h-5 w-5" />
              {loading || isStreaming ? 'Generating...' : 'Generate Quiz'}
            </button>
            
            {messages.length > 0 && (
              <button
                onClick={clearConversation}
                className="bg-gray-600 hover:bg-gray-700 text-white px-6 py-2 rounded-lg transition-colors"
              >
                Clear Conversation
              </button>
            )}
          </div>
        </div>

        {/* Quiz Conversation */}
        {messages.length > 0 && (
          <div className="border-t pt-6">
            <h3 className="text-lg font-semibold text-gray-700 mb-4">Quiz Conversation</h3>
            
            <div className="space-y-4 mb-6 max-h-96 overflow-y-auto">
              {messages
                .filter((message, index) => !(index === 0 && message.role === 'user' && message.content.startsWith('Generate quiz questions based on these notes')))
                .map((message, index) => (
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
              
              {(loading || isStreaming) && (
                <div className="bg-gray-50 border-l-4 border-gray-500 mr-8 p-4 rounded-lg">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-sm font-medium text-gray-700">🤖 AI Tutor</span>
                  </div>
                  {isStreaming && streamingContent ? (
                    <div className="text-gray-800 prose prose-sm max-w-none">
                      <ReactMarkdown>{streamingContent}</ReactMarkdown>
                      <div className="inline-block w-2 h-4 bg-gray-400 animate-pulse ml-1"></div>
                    </div>
                  ) : (
                    <div className="flex items-center gap-2">
                      <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-gray-600"></div>
                      <span className="text-gray-600">{isStreaming ? 'Generating response...' : 'Thinking...'}</span>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Message Input */}
            <form onSubmit={sendMessage} className="flex gap-4">
              <input
                type="text"
                value={currentMessage}
                onChange={(e) => setCurrentMessage(e.target.value)}
                placeholder="Ask a question or request more quiz questions..."
                className="flex-1 p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                disabled={loading || isStreaming}
              />
              <button
                type="submit"
                disabled={!currentMessage.trim() || loading || isStreaming}
                className="bg-purple-600 hover:bg-purple-700 disabled:bg-gray-400 text-white px-6 py-3 rounded-lg flex items-center gap-2 transition-colors"
              >
                <PaperAirplaneIcon className="h-5 w-5" />
                {loading || isStreaming ? 'Sending...' : 'Send'}
              </button>
            </form>
          </div>
        )}
      </div>
    </div>
  );
};

export default QuizGenerator;