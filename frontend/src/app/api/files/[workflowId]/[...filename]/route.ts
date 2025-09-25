import { NextRequest, NextResponse } from 'next/server';
import { exec } from 'child_process';
import { promisify } from 'util';
import path from 'path';
import fs from 'fs';

const execAsync = promisify(exec);

export async function GET(
  request: NextRequest,
  { params }: { params: { workflowId: string; filename: string[] } }
) {
  try {
    const workflowId = params.workflowId;
    const filename = params.filename.join('/');

    if (!workflowId || !filename) {
      return NextResponse.json(
        { error: 'Workflow ID and filename are required' },
        { status: 400 }
      );
    }

    // Find the project directory for this workflow
    const generatedDir = path.join(process.cwd(), '..', 'generated');
    let outputDir: string;

    try {
      const { stdout: lsOutput } = await execAsync(`ls -1t ${generatedDir} | grep project- | head -1`);
      const projectId = lsOutput.trim();

      if (!projectId) {
        return NextResponse.json({ error: 'No project directory found' }, { status: 404 });
      }

      outputDir = path.join(generatedDir, projectId);
      const filePath = path.join(outputDir, filename);

        // Security check - ensure the file is within the output directory
        const resolvedFilePath = path.resolve(filePath);
        const resolvedOutputDir = path.resolve(outputDir);

        if (!resolvedFilePath.startsWith(resolvedOutputDir)) {
          return NextResponse.json({ error: 'Invalid file path' }, { status: 403 });
        }

        // Read and return the file content
        try {
          const content = await fs.promises.readFile(filePath, 'utf-8');
          return new NextResponse(content, {
            headers: {
              'Content-Type': 'text/plain; charset=utf-8'
            }
          });
        } catch (fileError) {
          console.error('File read error:', fileError);
          return NextResponse.json({ error: 'File not found' }, { status: 404 });
        }

    } catch (cmdError) {
      console.error('Failed to get file:', cmdError);
      return NextResponse.json({ error: 'Failed to access file' }, { status: 500 });
    }

  } catch (error) {
    console.error('File API error:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}