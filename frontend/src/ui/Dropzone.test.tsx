import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Dropzone } from "./Dropzone";

function pdfFile(name = "doc.pdf", size = 1024) {
  return new File(["x".repeat(size)], name, { type: "application/pdf" });
}

function docxFile(name = "template.docx") {
  return new File(["x"], name, {
    type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  });
}

function txtFile(name = "notes.txt") {
  return new File(["x"], name, { type: "text/plain" });
}

describe("Dropzone", () => {
  it("renders label and hint", () => {
    render(<Dropzone files={[]} onFiles={vi.fn()} label="Upload" hint="PDF or DOCX" />);
    expect(screen.getByText("Upload")).toBeInTheDocument();
    expect(screen.getByText("PDF or DOCX")).toBeInTheDocument();
  });

  it("renders without hint", () => {
    render(<Dropzone files={[]} onFiles={vi.fn()} label="Upload" />);
    expect(screen.getByText("Upload")).toBeInTheDocument();
  });

  it("opens file dialog on click", async () => {
    const user = userEvent.setup();
    render(<Dropzone files={[]} onFiles={vi.fn()} label="Upload" />);

    const input = screen.getByTestId("dropzone-input") as HTMLInputElement;
    const clickSpy = vi.spyOn(input, "click");

    await user.click(screen.getByRole("button"));
    expect(clickSpy).toHaveBeenCalled();
  });

  it("opens file dialog on Enter key", () => {
    render(<Dropzone files={[]} onFiles={vi.fn()} label="Upload" />);

    const input = screen.getByTestId("dropzone-input") as HTMLInputElement;
    const clickSpy = vi.spyOn(input, "click");

    fireEvent.keyDown(screen.getByRole("button"), { key: "Enter" });
    expect(clickSpy).toHaveBeenCalled();
  });

  it("opens file dialog on Space key", () => {
    render(<Dropzone files={[]} onFiles={vi.fn()} label="Upload" />);

    const input = screen.getByTestId("dropzone-input") as HTMLInputElement;
    const clickSpy = vi.spyOn(input, "click");

    fireEvent.keyDown(screen.getByRole("button"), { key: " " });
    expect(clickSpy).toHaveBeenCalled();
  });

  it("does not open file dialog on other keys", () => {
    render(<Dropzone files={[]} onFiles={vi.fn()} label="Upload" />);

    const input = screen.getByTestId("dropzone-input") as HTMLInputElement;
    const clickSpy = vi.spyOn(input, "click");

    fireEvent.keyDown(screen.getByRole("button"), { key: "Tab" });
    expect(clickSpy).not.toHaveBeenCalled();
  });

  it("accepts PDF files via input", async () => {
    const onFiles = vi.fn();
    const user = userEvent.setup();
    render(<Dropzone files={[]} onFiles={onFiles} label="Upload" />);

    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, pdfFile());

    expect(onFiles).toHaveBeenCalledWith([expect.objectContaining({ name: "doc.pdf" })]);
  });

  it("accepts DOCX files via input", async () => {
    const onFiles = vi.fn();
    const user = userEvent.setup();
    render(<Dropzone files={[]} onFiles={onFiles} label="Upload" />);

    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, docxFile());

    expect(onFiles).toHaveBeenCalledWith([expect.objectContaining({ name: "template.docx" })]);
  });

  it("rejects non-PDF/DOCX files", async () => {
    const onFiles = vi.fn();
    const user = userEvent.setup();
    render(<Dropzone files={[]} onFiles={onFiles} label="Upload" />);

    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, txtFile());

    expect(onFiles).not.toHaveBeenCalled();
  });

  it("accepts files by extension when type is empty", async () => {
    const onFiles = vi.fn();
    const user = userEvent.setup();
    render(<Dropzone files={[]} onFiles={onFiles} label="Upload" />);

    const file = new File(["x"], "scan.pdf", { type: "" });
    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, file);

    expect(onFiles).toHaveBeenCalledWith([expect.objectContaining({ name: "scan.pdf" })]);
  });

  it("deduplicates files by name", async () => {
    const existing = [pdfFile("doc.pdf")];
    const onFiles = vi.fn();
    const user = userEvent.setup();
    render(<Dropzone files={existing} onFiles={onFiles} label="Upload" />);

    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, pdfFile("doc.pdf"));

    expect(onFiles).toHaveBeenCalledWith(existing);
  });

  it("shows file list with name and size", () => {
    const files = [pdfFile("doc.pdf", 2048), docxFile("tpl.docx")];
    render(<Dropzone files={files} onFiles={vi.fn()} label="Upload" />);

    expect(screen.getByText("doc.pdf")).toBeInTheDocument();
    expect(screen.getByText("2.0 KB")).toBeInTheDocument();
    expect(screen.getByText("tpl.docx")).toBeInTheDocument();
  });

  it("removes file on remove button click", async () => {
    const files = [pdfFile("a.pdf"), pdfFile("b.pdf")];
    const onFiles = vi.fn();
    const user = userEvent.setup();
    render(<Dropzone files={files} onFiles={onFiles} label="Upload" />);

    const removeBtn = screen.getByRole("button", { name: "Remove a.pdf" });
    await user.click(removeBtn);

    expect(onFiles).toHaveBeenCalledWith([expect.objectContaining({ name: "b.pdf" })]);
  });

  it("accepts files via drag and drop", () => {
    const onFiles = vi.fn();
    render(<Dropzone files={[]} onFiles={onFiles} label="Upload" />);

    const zone = screen.getByRole("button");

    fireEvent.dragOver(zone, { dataTransfer: { files: [] } });
    expect(zone.className).toContain("zoneDragOver");

    fireEvent.drop(zone, {
      dataTransfer: { files: [pdfFile("dropped.pdf")] },
    });

    expect(onFiles).toHaveBeenCalledWith([expect.objectContaining({ name: "dropped.pdf" })]);
  });

  it("removes drag over state on drag leave", () => {
    render(<Dropzone files={[]} onFiles={vi.fn()} label="Upload" />);

    const zone = screen.getByRole("button");

    fireEvent.dragOver(zone, { dataTransfer: { files: [] } });
    expect(zone.className).toContain("zoneDragOver");

    fireEvent.dragLeave(zone);
    expect(zone.className).not.toContain("zoneDragOver");
  });

  it("single mode replaces file", async () => {
    const onFiles = vi.fn();
    const user = userEvent.setup();
    render(
      <Dropzone files={[pdfFile("old.pdf")]} onFiles={onFiles} label="Upload" multiple={false} />,
    );

    const input = screen.getByTestId("dropzone-input");
    await user.upload(input, pdfFile("new.pdf"));

    expect(onFiles).toHaveBeenCalledWith([expect.objectContaining({ name: "new.pdf" })]);
  });

  it("formats bytes correctly", () => {
    const files = [
      new File(["x".repeat(500)], "small.pdf", { type: "application/pdf" }),
      new File(["x".repeat(1500000)], "big.pdf", { type: "application/pdf" }),
    ];
    render(<Dropzone files={files} onFiles={vi.fn()} label="Upload" />);

    expect(screen.getByText("500 B")).toBeInTheDocument();
    expect(screen.getByText("1.4 MB")).toBeInTheDocument();
  });
});
