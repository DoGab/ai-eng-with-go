import React, { useState, useEffect } from 'react';
import { PlusIcon, PencilIcon, TrashIcon } from '@heroicons/react/24/outline';
import { notesApi } from '../api/flashcardsApi';
import toast from 'react-hot-toast';

const NotesManager = () => {
  const [notes, setNotes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [newNote, setNewNote] = useState('');
  const [editingNote, setEditingNote] = useState(null);
  const [editContent, setEditContent] = useState('');

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
      setLoading(false);
    }
  };

  const handleCreateNote = async (e) => {
    e.preventDefault();
    if (!newNote.trim()) return;

    try {
      const response = await notesApi.create({ content: newNote });
      setNotes([...notes, response.data]);
      setNewNote('');
      toast.success('Note created successfully!');
    } catch (error) {
      toast.error('Failed to create note');
      console.error('Error creating note:', error);
    }
  };

  const handleUpdateNote = async (id) => {
    if (!editContent.trim()) return;

    try {
      const response = await notesApi.update(id, { content: editContent });
      setNotes(notes.map(note => note.id === id ? response.data : note));
      setEditingNote(null);
      setEditContent('');
      toast.success('Note updated successfully!');
    } catch (error) {
      toast.error('Failed to update note');
      console.error('Error updating note:', error);
    }
  };

  const handleDeleteNote = async (id) => {
    if (!window.confirm('Are you sure you want to delete this note?')) return;

    try {
      await notesApi.delete(id);
      setNotes(notes.filter(note => note.id !== id));
      toast.success('Note deleted successfully!');
    } catch (error) {
      toast.error('Failed to delete note');
      console.error('Error deleting note:', error);
    }
  };

  const startEditing = (note) => {
    setEditingNote(note.id);
    setEditContent(note.content);
  };

  const cancelEditing = () => {
    setEditingNote(null);
    setEditContent('');
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="bg-white rounded-lg shadow-lg p-6 mb-8">
        <h2 className="text-2xl font-bold text-gray-800 mb-6">📚 Notes Manager</h2>
        
        {/* Create new note form */}
        <form onSubmit={handleCreateNote} className="mb-6">
          <div className="flex gap-4">
            <textarea
              value={newNote}
              onChange={(e) => setNewNote(e.target.value)}
              placeholder="Write your note here..."
              className="flex-1 p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
              rows="3"
            />
            <button
              type="submit"
              className="bg-blue-600 hover:bg-blue-700 text-white px-6 py-3 rounded-lg flex items-center gap-2 transition-colors"
            >
              <PlusIcon className="h-5 w-5" />
              Add Note
            </button>
          </div>
        </form>

        {/* Notes list */}
        <div className="space-y-4">
          {notes.length === 0 ? (
            <div className="text-center py-12 text-gray-500">
              <p className="text-lg">No notes yet. Create your first note above!</p>
            </div>
          ) : (
            notes.map((note) => (
              <div key={note.id} className="bg-gray-50 rounded-lg p-4 border border-gray-200">
                {editingNote === note.id ? (
                  <div className="space-y-3">
                    <textarea
                      value={editContent}
                      onChange={(e) => setEditContent(e.target.value)}
                      className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
                      rows="3"
                    />
                    <div className="flex gap-2">
                      <button
                        onClick={() => handleUpdateNote(note.id)}
                        className="bg-green-600 hover:bg-green-700 text-white px-4 py-2 rounded-md text-sm transition-colors"
                      >
                        Save
                      </button>
                      <button
                        onClick={cancelEditing}
                        className="bg-gray-600 hover:bg-gray-700 text-white px-4 py-2 rounded-md text-sm transition-colors"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="flex justify-between items-start">
                    <div className="flex-1">
                      <p className="text-gray-800 whitespace-pre-wrap">{note.content}</p>
                      <p className="text-sm text-gray-500 mt-2">
                        Created: {new Date(note.createdAt).toLocaleDateString()} | 
                        Updated: {new Date(note.updatedAt).toLocaleDateString()}
                      </p>
                    </div>
                    <div className="flex gap-2 ml-4">
                      <button
                        onClick={() => startEditing(note)}
                        className="text-blue-600 hover:text-blue-800 p-1 rounded-md transition-colors"
                        title="Edit note"
                      >
                        <PencilIcon className="h-5 w-5" />
                      </button>
                      <button
                        onClick={() => handleDeleteNote(note.id)}
                        className="text-red-600 hover:text-red-800 p-1 rounded-md transition-colors"
                        title="Delete note"
                      >
                        <TrashIcon className="h-5 w-5" />
                      </button>
                    </div>
                  </div>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
};

export default NotesManager;