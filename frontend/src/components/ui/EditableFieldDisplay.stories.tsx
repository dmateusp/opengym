import { EditableFieldDisplay } from './EditableFieldDisplay';

export const Default = () => (
  <EditableFieldDisplay
    isEditing={false}
    isEditable={true}
  >
    <span className="text-lg font-semibold text-gray-900">Sample Field Text</span>
  </EditableFieldDisplay>
);

export const NotEditable = () => (
  <EditableFieldDisplay
    isEditing={false}
    isEditable={false}
  >
    <span className="text-lg font-semibold text-gray-900">Read-only Field</span>
  </EditableFieldDisplay>
);

export const Editing = () => (
  <EditableFieldDisplay
    isEditing={true}
    isEditable={true}
  >
    <span className="text-lg font-semibold text-gray-900">Currently Editing</span>
  </EditableFieldDisplay>
);

export const WithPlaceholder = () => (
  <EditableFieldDisplay
    isEditing={false}
    isEditable={true}
  >
    <span className="text-lg font-semibold text-gray-400">Click to add content...</span>
  </EditableFieldDisplay>
);

export const InteractiveExample = () => (
  <EditableFieldDisplay
    isEditing={false}
    isEditable={true}
    onClick={() => alert('Field clicked! Would open edit mode.')}
  >
    <span className="text-lg font-semibold text-gray-900">Click me to edit</span>
  </EditableFieldDisplay>
);
