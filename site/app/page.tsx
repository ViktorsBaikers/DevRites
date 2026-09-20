import Nav from "@/components/Nav";
import Hero from "@/components/Hero";
import RegisterSections from "@/components/RegisterSections";

export default function Home() {
  return (
    <>
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-surface focus:px-4 focus:py-2 focus:text-ink"
      >
        Skip to content
      </a>
      <Nav />
      <main id="main" className="w-full max-w-full overflow-x-clip">
        <Hero />
        <RegisterSections />
      </main>
    </>
  );
}
