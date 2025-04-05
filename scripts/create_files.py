import sys
import os

def create_files_from_message(message_content):
    """
    Parses a message containing file contents and creates files on disk.
    Assumes message is split by '---' and each file section starts with the filepath.
    """
    file_sections = message_content.strip().split("---")

    for section in file_sections:
        section = section.strip()
        if not section:  # Skip empty sections
            continue

        lines = section.strip().splitlines()
        if not lines: # Skip sections with no lines
            continue

        filepath = lines[0].strip() # First line is the filepath

        if not filepath:
            print(f"Warning: Section without filepath, skipping section starting with: {section[:50]}...")
            continue

        file_content = "\n".join(lines[1:]).strip() # Rest of lines is content

        dirname = os.path.dirname(filepath)
        if dirname: # Only create dirs if there's a dirname component.
            os.makedirs(dirname, exist_ok=True)

        with open(filepath, 'w') as f:
            f.write(file_content)
        print(f"Created file: {filepath}")

if __name__ == "__main__":
    message_input = sys.stdin.read()
    create_files_from_message(message_input)
    print("\nFile creation process completed.")
